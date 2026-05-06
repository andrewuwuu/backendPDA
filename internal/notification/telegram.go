package notification

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"pda-monitor/internal/domain"
	"pda-monitor/internal/logger"
	"pda-monitor/internal/timeutil"
	"pda-monitor/internal/util"
)

const component = "Telegram"

type TelegramConfig struct {
	BotToken string
	ChatIDs  []int64
	Channels []string
}

type TelegramNotifier struct {
	bot      *tgbotapi.BotAPI
	chatIDs  []int64
	channels []string
}

func NewTelegramNotifier(config TelegramConfig) (*TelegramNotifier, error) {
	bot, err := tgbotapi.NewBotAPI(config.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	logger.Info(component, "Telegram bot initialized", logger.F("bot_username", bot.Self.UserName))

	return &TelegramNotifier{
		bot:      bot,
		chatIDs:  config.ChatIDs,
		channels: config.Channels,
	}, nil
}

func (t *TelegramNotifier) SendDebitReport(ctx context.Context, results []domain.DebitResult) error {
	message := t.formatDebitReport(results)
	return t.sendMessage(message)
}

func (t *TelegramNotifier) SendDailyExcelReport(ctx context.Context, reports []domain.DailyStationReport, excelData *bytes.Buffer, filename string) error {
	message := t.formatDailyReport(reports)
	if err := t.sendMessage(message); err != nil {
		return fmt.Errorf("failed to send text report: %w", err)
	}

	if err := t.sendDocument(excelData, filename); err != nil {
		return fmt.Errorf("failed to send excel file: %w", err)
	}

	return nil
}

func (t *TelegramNotifier) sendMessage(message string) error {
	successCount := 0
	failCount := 0

	for _, chatID := range t.chatIDs {
		msg := tgbotapi.NewMessage(chatID, message)
		msg.ParseMode = tgbotapi.ModeHTML

		if _, err := t.bot.Send(msg); err != nil {
			logger.Error(component, "Failed to send message to chat", logger.Fields(
				"chat_id", chatID,
				"error", err.Error(),
			))
			failCount++
		} else {
			successCount++
		}
	}

	for _, channel := range t.channels {
		msg := tgbotapi.NewMessageToChannel(channel, message)
		msg.ParseMode = tgbotapi.ModeHTML

		if _, err := t.bot.Send(msg); err != nil {
			logger.Error(component, "Failed to send message to channel", logger.Fields(
				"channel", channel,
				"error", err.Error(),
			))
			failCount++
		} else {
			successCount++
		}
	}

	logger.Debug(component, "Message delivery completed", logger.Fields(
		"success", successCount,
		"failed", failCount,
	))

	if failCount > 0 && successCount == 0 {
		return fmt.Errorf("failed to send to all %d recipients", failCount)
	}

	return nil
}

func (t *TelegramNotifier) sendDocument(data *bytes.Buffer, filename string) error {
	fileBytes := tgbotapi.FileBytes{
		Name:  filename,
		Bytes: data.Bytes(),
	}

	successCount := 0
	failCount := 0

	for _, chatID := range t.chatIDs {
		doc := tgbotapi.NewDocument(chatID, fileBytes)
		doc.Caption = "Laporan Harian Debit PDA"

		if _, err := t.bot.Send(doc); err != nil {
			logger.Error(component, "Failed to send document to chat", logger.Fields(
				"chat_id", chatID,
				"filename", filename,
				"error", err.Error(),
			))
			failCount++
		} else {
			successCount++
		}
	}

	for _, channel := range t.channels {
		doc := tgbotapi.DocumentConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{
					ChannelUsername: channel,
				},
				File: fileBytes,
			},
			Caption: "Laporan Harian Debit PDA",
		}

		if _, err := t.bot.Send(doc); err != nil {
			logger.Error(component, "Failed to send document to channel", logger.Fields(
				"channel", channel,
				"filename", filename,
				"error", err.Error(),
			))
			failCount++
		} else {
			successCount++
		}
	}

	logger.Debug(component, "Document delivery completed", logger.Fields(
		"filename", filename,
		"size_bytes", data.Len(),
		"success", successCount,
		"failed", failCount,
	))

	if failCount > 0 && successCount == 0 {
		return fmt.Errorf("failed to send document to all %d recipients", failCount)
	}

	return nil
}

func (t *TelegramNotifier) formatDebitReport(results []domain.DebitResult) string {
	now := timeutil.NowJakarta()

	var sb strings.Builder

	sb.WriteString("<b>LAPORAN DEBIT PDA</b>\n")
	sb.WriteString(fmt.Sprintf("%s\n", now.Format("02 Jan 2006, 15:04 WIB")))
	sb.WriteString(strings.Repeat("─", 25) + "\n\n")

	for _, r := range results {
		displayName := util.CleanStationName(r.NamaLokasi, r.NamaAlat)

		if r.IsValid {
			sb.WriteString(fmt.Sprintf("%s | %.3f m³/s\n", displayName, r.Debit))
		} else {
			sb.WriteString(fmt.Sprintf("%s | -\n", displayName))
		}
	}

	sb.WriteString("\n" + strings.Repeat("─", 25))

	return sb.String()
}

func (t *TelegramNotifier) formatDailyReport(reports []domain.DailyStationReport) string {
	now := timeutil.NowJakarta()

	var sb strings.Builder

	sb.WriteString("<b>LAPORAN HARIAN DEBIT PDA</b>\n")
	sb.WriteString(fmt.Sprintf("%s\n", now.Format("02 Jan 2006")))
	sb.WriteString(strings.Repeat("─", 30) + "\n\n")

	sb.WriteString("<pre>")
	sb.WriteString(fmt.Sprintf("%-3s | %-20s | %7s | %7s | %7s | %6s | %6s\n",
		"No", "Nama PDA", "07:00", "12:00", "17:00", "TMA↓", "TMA↑"))
	sb.WriteString(strings.Repeat("-", 75) + "\n")

	for i, r := range reports {
		debit07 := "-"
		debit12 := "-"
		debit17 := "-"
		minTMA := "-"
		maxTMA := "-"

		if r.Debit07 != nil {
			debit07 = fmt.Sprintf("%.2f", *r.Debit07)
		}
		if r.Debit12 != nil {
			debit12 = fmt.Sprintf("%.2f", *r.Debit12)
		}
		if r.Debit17 != nil {
			debit17 = fmt.Sprintf("%.2f", *r.Debit17)
		}
		if r.MinTMA != nil {
			minTMA = fmt.Sprintf("%.2f", *r.MinTMA)
		}
		if r.MaxTMA != nil {
			maxTMA = fmt.Sprintf("%.2f", *r.MaxTMA)
		}

		name := r.NamaAlat
		if len(name) > 20 {
			name = name[:17] + "..."
		}

		sb.WriteString(fmt.Sprintf("%-3d | %-20s | %7s | %7s | %7s | %6s | %6s\n",
			i+1, name, debit07, debit12, debit17, minTMA, maxTMA))
	}

	sb.WriteString("</pre>\n")
	sb.WriteString("\n" + strings.Repeat("─", 30))
	sb.WriteString("\n<i>Debit dalam m³/s, TMA dalam meter</i>")

	return sb.String()
}

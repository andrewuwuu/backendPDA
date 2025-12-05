package notification

import (
    "context"
    "fmt"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

    "pda-monitor/internal/domain"
)

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

    return &TelegramNotifier{
        bot:      bot,
        chatIDs:  config.ChatIDs,
        channels: config.Channels,
    }, nil
}

func (t *TelegramNotifier) SendDebitReport(ctx context.Context, results []domain.DebitResult) error {
    message := t.formatDebitReport(results)

    // Send to numeric chat IDs
    for _, chatID := range t.chatIDs {
        msg := tgbotapi.NewMessage(chatID, message)
        msg.ParseMode = tgbotapi.ModeHTML

        if _, err := t.bot.Send(msg); err != nil {
            return fmt.Errorf("failed to send to chat %d: %w", chatID, err)
        }
    }

    // Send to @channel_username
    for _, channel := range t.channels {
        msg := tgbotapi.NewMessageToChannel(channel, message)
        msg.ParseMode = tgbotapi.ModeHTML

        if _, err := t.bot.Send(msg); err != nil {
            return fmt.Errorf("failed to send to channel %s: %w", channel, err)
        }
    }

    return nil
}

func (t *TelegramNotifier) formatDebitReport(results []domain.DebitResult) string {
    now := time.Now()

    var sb strings.Builder

    // Header
    sb.WriteString(fmt.Sprintf("<b>LAPORAN DEBIT PDA</b>\n"))
    sb.WriteString(fmt.Sprintf("%s\n", now.Format("02 Jan 2006, 15:04 WIB")))
    sb.WriteString(strings.Repeat("─", 25) + "\n\n")

    // Simple list: Station | Debit
    for _, r := range results {
        if r.IsValid {
            sb.WriteString(fmt.Sprintf("%s | %.3f m³/s\n", r.NamaAlat, r.Debit))
        } else {
            sb.WriteString(fmt.Sprintf("%s | -\n", r.NamaAlat))
        }
    }

    sb.WriteString("\n" + strings.Repeat("─", 25))

    return sb.String()
}
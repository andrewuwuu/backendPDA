package config

import (
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"

    "github.com/joho/godotenv"
)

type Config struct {
    Server    ServerConfig
    Database  DatabaseConfig
    Telemetry TelemetryConfig
    Telegram  TelegramConfig
}

type ServerConfig struct {
    Port string
}

type DatabaseConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    Name     string
}

func (d DatabaseConfig) DSN() string {
    return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
        d.User,
        d.Password,
        d.Host,
        d.Port,
        d.Name,
    )
}

type TelemetryConfig struct {
    BaseURL    string
    IDBBWS     string
    TimeoutSec int
}

type TelegramConfig struct {
    BotToken string
    ChatIDs  []int64
}

func Load() *Config {
    if err := godotenv.Load(); err != nil {
        log.Println("Warning: .env file not found, using environment variables")
    }

    cfg := &Config{
        Server: ServerConfig{
            Port: getEnv("PORT", "8080"),
        },
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnv("DB_PORT", "3306"),
            User:     getEnv("DB_USER", "root"),
            Password: getEnv("DB_PASSWORD", ""),
            Name:     getEnv("DB_NAME", "pda_monitor"),
        },
        Telemetry: TelemetryConfig{
            BaseURL:    getEnv("TELEMETRY_BASE_URL", ""),
            IDBBWS:     getEnv("TELEMETRY_IDBBWS", ""),
            TimeoutSec: getEnvInt("TELEMETRY_TIMEOUT", 30),
        },
        Telegram: TelegramConfig{
            BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
            ChatIDs:  getEnvInt64Slice("TELEGRAM_CHAT_IDS"),
        },
    }

    cfg.validate()
    return cfg
}

func (c *Config) validate() {
    if c.Telemetry.BaseURL == "" {
        log.Fatal("TELEMETRY_BASE_URL is required")
    }
    if c.Telemetry.IDBBWS == "" {
        log.Fatal("TELEMETRY_IDBBWS is required")
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return defaultValue
}

func getEnvInt64Slice(key string) []int64 {
    value := os.Getenv(key)
    if value == "" {
        return nil
    }

    parts := strings.Split(value, ",")
    result := make([]int64, 0, len(parts))

    for _, p := range parts {
        p = strings.TrimSpace(p)
        if id, err := strconv.ParseInt(p, 10, 64); err == nil {
            result = append(result, id)
        }
    }

    return result
}
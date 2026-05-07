package config

import (
	"fmt"
	"log"
	"net/url"
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
	JWT       JWTConfig
	Logging   LoggingConfig
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
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=%s",
		d.User,
		d.Password,
		d.Host,
		d.Port,
		d.Name,
		url.QueryEscape("Asia/Jakarta"),
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
	Channels []string
}

type JWTConfig struct {
	ExpiryHours int
}

type LoggingConfig struct {
	Level    string
	FilePath string
	Console  bool
}

func Load() *Config {
	return LoadServer()
}

func LoadServer() *Config {
	cfg := load("")
	cfg.validateServer()
	return cfg
}

func LoadDatabase() *Config {
	return load("")
}

func LoadDatabaseFromEnvFile(path string) *Config {
	return load(path)
}

func load(envFile string) *Config {
	if envFile != "" {
		if err := godotenv.Overload(envFile); err != nil {
			log.Printf("Warning: failed to load env file %s, using current environment: %v", envFile, err)
		}
	} else if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	return &Config{
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
			Channels: getEnvStringSlice("TELEGRAM_CHANNELS"),
		},
		JWT: JWTConfig{
			ExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 24),
		},
		Logging: LoggingConfig{
			Level:    getEnv("LOG_LEVEL", "INFO"),
			FilePath: getEnv("LOG_FILE", ""),
			Console:  getEnvBool("LOG_CONSOLE", true),
		},
	}
}

func (c *Config) validateServer() {
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
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

func getEnvStringSlice(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}

	return result
}

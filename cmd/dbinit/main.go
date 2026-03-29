package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"pda-monitor/internal/config"
	"pda-monitor/internal/database"
	"pda-monitor/internal/domain"
	"pda-monitor/internal/logger"
	mysqlrepo "pda-monitor/internal/repository/mysql"
)

const component = "DBInit"

func main() {
	envFile := flag.String("env-file", "", "Path to env file. Defaults to loading .env from the current working directory.")
	dsnOverride := flag.String("dsn", "", "MySQL DSN override. Defaults to DB_* environment variables.")
	timeout := flag.Duration("timeout", 15*time.Second, "Database connection timeout.")
	createUser := flag.Bool("create-user", false, "Create an API user after schema initialization.")
	username := flag.String("username", "", "Username for the API user.")
	password := flag.String("password", "", "Password for the API user.")
	role := flag.String("role", "admin", "Role for the API user: admin or user.")
	flag.Parse()

	cfg := config.LoadDatabaseFromEnvFile(*envFile)
	if err := logger.Init(logger.Config{
		Level:    cfg.Logging.Level,
		FilePath: cfg.Logging.FilePath,
		Console:  cfg.Logging.Console,
	}); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Close()

	dsn := cfg.Database.DSN()
	if *dsnOverride != "" {
		dsn = *dsnOverride
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		logger.Fatal(component, "Failed to connect to database", logger.F("error", err.Error()))
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		logger.Fatal(component, "Failed to connect to database", logger.F("error", err.Error()))
	}

	if err := database.InitSchema(ctx, db); err != nil {
		logger.Fatal(component, "Failed to initialize schema", logger.F("error", err.Error()))
	}

	tables := strings.Join(database.TableNames(), ", ")
	logger.Info(component, "Schema initialized successfully", logger.F("tables", tables))
	fmt.Fprintf(os.Stdout, "Initialized tables: %s\n", tables)

	if !*createUser {
		return
	}

	if err := createAPIUser(ctx, db, *username, *password, *role); err != nil {
		logger.Fatal(component, "Failed to create API user", logger.F("error", err.Error()))
	}
}

func createAPIUser(ctx context.Context, db *sqlx.DB, username, password, role string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	role = strings.ToLower(strings.TrimSpace(role))

	if username == "" {
		return fmt.Errorf("username is required when -create-user is set")
	}
	if password == "" {
		return fmt.Errorf("password is required when -create-user is set")
	}
	if role != "admin" && role != "user" {
		return fmt.Errorf("role must be either admin or user")
	}

	userRepo := mysqlrepo.NewUserRepo(db)
	existing, err := userRepo.GetByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("user %q already exists", username)
	}

	user := &domain.User{
		Username: username,
		Role:     role,
	}
	if err := userRepo.Create(ctx, user, password); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	logger.Info(component, "API user created successfully", logger.Fields(
		"user_id", user.ID,
		"username", user.Username,
		"role", user.Role,
	))
	fmt.Fprintf(os.Stdout, "Created API user: %s (%s)\n", user.Username, user.Role)
	return nil
}

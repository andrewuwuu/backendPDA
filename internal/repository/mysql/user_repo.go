package mysql

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/jmoiron/sqlx"
    "golang.org/x/crypto/bcrypt"

    "pda-monitor/internal/domain"
    "pda-monitor/internal/logger"
)

const component = "UserRepo"

type UserRepo struct {
    db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
    return &UserRepo{db: db}
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
    var user domain.User
    query := `SELECT * FROM users WHERE username = ?`
    err := r.db.GetContext(ctx, &user, query, username)
    if err == sql.ErrNoRows {
        logger.Debug(component, "User not found", logger.F("username", username))
        return nil, nil
    }
    if err != nil {
        logger.Error(component, "Failed to get user by username", logger.Fields(
            "username", username,
            "error", err.Error(),
        ))
        return nil, err
    }
    return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
    var user domain.User
    query := `SELECT * FROM users WHERE id = ?`
    err := r.db.GetContext(ctx, &user, query, id)
    if err == sql.ErrNoRows {
        logger.Debug(component, "User not found", logger.F("user_id", id))
        return nil, nil
    }
    if err != nil {
        logger.Error(component, "Failed to get user by ID", logger.Fields(
            "user_id", id,
            "error", err.Error(),
        ))
        return nil, err
    }
    return &user, nil
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User, password string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        logger.Error(component, "Failed to hash password", logger.Fields(
            "username", user.Username,
            "error", err.Error(),
        ))
        return fmt.Errorf("failed to hash password: %w", err)
    }

    query := `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`
    result, err := r.db.ExecContext(ctx, query, user.Username, string(hash), user.Role)
    if err != nil {
        logger.Error(component, "Failed to create user", logger.Fields(
            "username", user.Username,
            "role", user.Role,
            "error", err.Error(),
        ))
        return fmt.Errorf("failed to create user: %w", err)
    }

    id, _ := result.LastInsertId()
    user.ID = id

    logger.Info(component, "User created successfully", logger.Fields(
        "user_id", id,
        "username", user.Username,
        "role", user.Role,
    ))

    return nil
}

func (r *UserRepo) ValidatePassword(user *domain.User, password string) bool {
    if user == nil {
        logger.Warn(component, "Password validation attempted with nil user")
        return false
    }

    err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
    if err != nil {
        logger.Warn(component, "Password validation failed", logger.Fields(
            "user_id", user.ID,
            "username", user.Username,
        ))
        return false
    }

    logger.Debug(component, "Password validation successful", logger.Fields(
        "user_id", user.ID,
        "username", user.Username,
    ))

    return true
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
    if err != nil {
        logger.Error(component, "Failed to hash new password", logger.Fields(
            "user_id", userID,
            "error", err.Error(),
        ))
        return fmt.Errorf("failed to hash password: %w", err)
    }

    query := `UPDATE users SET password_hash = ? WHERE id = ?`
    result, err := r.db.ExecContext(ctx, query, string(hash), userID)
    if err != nil {
        logger.Error(component, "Failed to update password", logger.Fields(
            "user_id", userID,
            "error", err.Error(),
        ))
        return fmt.Errorf("failed to update password: %w", err)
    }

    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        logger.Warn(component, "Password update affected no rows", logger.F("user_id", userID))
        return fmt.Errorf("user not found")
    }

    logger.Info(component, "Password updated successfully", logger.F("user_id", userID))
    return nil
}
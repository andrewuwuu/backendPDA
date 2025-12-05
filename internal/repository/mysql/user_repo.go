package mysql

import (
    "context"
    "database/sql"

    "github.com/jmoiron/sqlx"
    "golang.org/x/crypto/bcrypt"

    "pda-monitor/internal/domain"
)

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
        return nil, nil
    }
    return &user, err
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
    var user domain.User
    query := `SELECT * FROM users WHERE id = ?`
    err := r.db.GetContext(ctx, &user, query, id)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &user, err
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User, password string) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    query := `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`
    result, err := r.db.ExecContext(ctx, query, user.Username, string(hash), user.Role)
    if err != nil {
        return err
    }

    id, _ := result.LastInsertId()
    user.ID = id
    return nil
}

func (r *UserRepo) ValidatePassword(user *domain.User, password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
    return err == nil
}
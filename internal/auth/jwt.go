package auth

import (
    "crypto/rand"
    "encoding/base64"
    "errors"
    "log"
    "sync"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

var (
    ErrInvalidToken = errors.New("invalid token")
    ErrExpiredToken = errors.New("token has expired")
)

type Claims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

type JWTManager struct {
    currentKey  []byte
    previousKey []byte
    expiryHours int
    mu          sync.RWMutex
    stopChan    chan struct{}
}

// NewJWTManager creates a new JWT manager with auto-generated 256-bit key
func NewJWTManager(expiryHours int) *JWTManager {
    m := &JWTManager{
        currentKey:  generateKey(),
        previousKey: nil,
        expiryHours: expiryHours,
        stopChan:    make(chan struct{}),
    }

    log.Printf("[JWT] Initial key generated (256-bit)")
    log.Printf("[JWT] Key rotation scheduled every 24 hours")

    // Start key rotation goroutine
    go m.startKeyRotation()

    return m
}

// generateKey creates a cryptographically secure 256-bit (32 byte) key
func generateKey() []byte {
    key := make([]byte, 32) // 256 bits
    if _, err := rand.Read(key); err != nil {
        log.Fatalf("Failed to generate JWT key: %v", err)
    }
    return key
}

// startKeyRotation rotates key every 24 hours
func (m *JWTManager) startKeyRotation() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            m.rotateKey()
        case <-m.stopChan:
            return
        }
    }
}

// rotateKey performs key rotation
func (m *JWTManager) rotateKey() {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.previousKey = m.currentKey
    m.currentKey = generateKey()

    log.Printf("[JWT] Key rotated at %s", time.Now().Format(time.RFC3339))
}

// Stop stops the key rotation goroutine
func (m *JWTManager) Stop() {
    close(m.stopChan)
}

// GenerateToken creates a new JWT token
func (m *JWTManager) GenerateToken(userID int64, username, role string) (string, error) {
    m.mu.RLock()
    key := m.currentKey
    m.mu.RUnlock()

    claims := &Claims{
        UserID:   userID,
        Username: username,
        Role:     role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(m.expiryHours) * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(key)
}

// ValidateToken validates and parses a JWT token
// Tries current key first, then previous key (for rotation grace period)
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    m.mu.RLock()
    currentKey := m.currentKey
    previousKey := m.previousKey
    m.mu.RUnlock()

    // Try current key first
    claims, err := m.validateWithKey(tokenString, currentKey)
    if err == nil {
        return claims, nil
    }

    // Try previous key (grace period after rotation)
    if previousKey != nil {
        claims, err = m.validateWithKey(tokenString, previousKey)
        if err == nil {
            return claims, nil
        }
    }

    return nil, ErrInvalidToken
}

func (m *JWTManager) validateWithKey(tokenString string, key []byte) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidToken
        }
        return key, nil
    })

    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrInvalidToken
    }

    return claims, nil
}

// GetKeyInfo returns key info for debugging (not the actual key)
func (m *JWTManager) GetKeyInfo() map[string]interface{} {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return map[string]interface{}{
        "current_key_hash":  base64.StdEncoding.EncodeToString(m.currentKey[:8]) + "...",
        "has_previous_key":  m.previousKey != nil,
        "expiry_hours":      m.expiryHours,
        "rotation_interval": "24h",
    }
}

// ForceRotate manually triggers key rotation (for testing)
func (m *JWTManager) ForceRotate() {
    m.rotateKey()
}
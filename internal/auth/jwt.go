package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"pda-monitor/internal/logger"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrBlacklistedToken = errors.New("token has been invalidated")
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type blacklistEntry struct {
	expiresAt time.Time
}

type JWTManager struct {
	currentKey  []byte
	previousKey []byte
	expiryHours int
	mu          sync.RWMutex
	stopChan    chan struct{}
	blacklist   map[string]blacklistEntry
	blacklistMu sync.RWMutex
}

func NewJWTManager(expiryHours int) *JWTManager {
	m := &JWTManager{
		currentKey:  generateKey(),
		previousKey: nil,
		expiryHours: expiryHours,
		stopChan:    make(chan struct{}),
		blacklist:   make(map[string]blacklistEntry),
	}

	logger.Info("JWT", "Manager initialized", logger.Fields(
		"key_size", "256-bit",
		"rotation_interval", "24h",
	))

	// Start key rotation goroutine
	go m.startKeyRotation()
	go m.startBlacklistCleanup()

	return m
}

func generateKey() []byte {
	key := make([]byte, 32) // 256 bits
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate JWT key: %v", err)
	}
	return key
}

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

func (m *JWTManager) rotateKey() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.previousKey = m.currentKey
	m.currentKey = generateKey()

	logger.Info("JWT", "Key rotated successfully")
}

func (m *JWTManager) Stop() {
	close(m.stopChan)
}

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

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	if m.IsBlacklisted(tokenString) {
		return nil, ErrBlacklistedToken
	}

	m.mu.RLock()
	currentKey := m.currentKey
	previousKey := m.previousKey
	m.mu.RUnlock()

	claims, err := m.validateWithKey(tokenString, currentKey)
	if err == nil {
		return claims, nil
	}

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

func (m *JWTManager) ForceRotate() {
	m.rotateKey()
}

func (m *JWTManager) startBlacklistCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupBlacklist()
		case <-m.stopChan:
			return
		}
	}
}

func (m *JWTManager) cleanupBlacklist() {
	m.blacklistMu.Lock()
	defer m.blacklistMu.Unlock()

	now := time.Now()
	for token, entry := range m.blacklist {
		if now.After(entry.expiresAt) {
			delete(m.blacklist, token)
		}
	}
}

func (m *JWTManager) InvalidateToken(tokenString string) error {
	claims, err := m.validateTokenWithoutBlacklistCheck(tokenString)
	if err != nil {
		return err
	}

	m.blacklistMu.Lock()
	defer m.blacklistMu.Unlock()

	m.blacklist[tokenString] = blacklistEntry{
		expiresAt: claims.ExpiresAt.Time,
	}

	return nil
}

func (m *JWTManager) validateTokenWithoutBlacklistCheck(tokenString string) (*Claims, error) {
	m.mu.RLock()
	currentKey := m.currentKey
	previousKey := m.previousKey
	m.mu.RUnlock()

	claims, err := m.validateWithKey(tokenString, currentKey)
	if err == nil {
		return claims, nil
	}

	if previousKey != nil {
		claims, err = m.validateWithKey(tokenString, previousKey)
		if err == nil {
			return claims, nil
		}
	}

	return nil, ErrInvalidToken
}

func (m *JWTManager) IsBlacklisted(tokenString string) bool {
	m.blacklistMu.RLock()
	defer m.blacklistMu.RUnlock()

	_, exists := m.blacklist[tokenString]
	return exists
}

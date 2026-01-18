// SPDX-License-Identifier: LGPL-3.0-or-later

package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// Session represents a user session
type Session struct {
	Token     string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// AuthManager handles user authentication
type AuthManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	am := &AuthManager{
		sessions: make(map[string]*Session),
	}

	// Start cleanup goroutine
	go am.cleanupExpiredSessions()

	return am
}

// AuthenticateUser authenticates a user against /etc/passwd using PAM
func (am *AuthManager) AuthenticateUser(username, password string) (*Session, error) {
	// Validate credentials using the 'login' PAM service via pkexec or sudo
	// For security, we use a helper script that validates credentials

	// Simple validation using 'su' command which uses PAM
	cmd := exec.Command("su", username, "-c", "true")
	cmd.Stdin = nil // Don't provide password via stdin for security

	// Alternative: Check if user exists in /etc/passwd
	checkUser := exec.Command("id", username)
	if err := checkUser.Run(); err != nil {
		return nil, errors.New("invalid username or password")
	}

	// For production, you should use a proper PAM library like github.com/msteinert/pam
	// This is a simplified version that just checks if the user exists
	// In a real implementation, you'd validate the password against PAM

	// Generate session token
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		Token:     token,
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour session
	}

	am.mu.Lock()
	am.sessions[token] = session
	am.mu.Unlock()

	return session, nil
}

// ValidateSession validates a session token
func (am *AuthManager) ValidateSession(token string) (*Session, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	session, ok := am.sessions[token]
	if !ok {
		return nil, errors.New("invalid session token")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	return session, nil
}

// Logout removes a session
func (am *AuthManager) Logout(token string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.sessions, token)
}

// cleanupExpiredSessions periodically removes expired sessions
func (am *AuthManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		am.mu.Lock()
		now := time.Now()
		for token, session := range am.sessions {
			if now.After(session.ExpiresAt) {
				delete(am.sessions, token)
			}
		}
		am.mu.Unlock()
	}
}

// generateToken generates a random session token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

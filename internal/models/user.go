package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Wallets   []Wallet  `json:"wallets,omitempty"`
	Emails    []Email   `json:"emails,omitempty"`
}

// Wallet represents a crypto wallet
type Wallet struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Address   string    `json:"address" db:"address"`
	Type      string    `json:"type" db:"type"` // e.g., "bitcoin", "ethereum"
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Email represents an email address associated with a user
type Email struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Address   string    `json:"address" db:"address"`
	Verified  bool      `json:"verified" db:"verified"`
	Primary   bool      `json:"primary" db:"primary"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// EmailVerification represents an email verification record
type EmailVerification struct {
	ID        string    `json:"id" db:"id"`
	EmailID   string    `json:"email_id" db:"email_id"`
	Code      string    `json:"code" db:"code"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AuthToken represents the authentication token response
type AuthToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user,omitempty"`
}

// WalletAuthRequest represents a request to authenticate with a wallet
type WalletAuthRequest struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
	Message   string `json:"message"`
}

// EmailAuthRequest represents a request to authenticate with an email
type EmailAuthRequest struct {
	Email string `json:"email"`
}

// EmailVerifyRequest represents a request to verify an email code
type EmailVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
} 
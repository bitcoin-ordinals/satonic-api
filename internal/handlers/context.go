package handlers

import (
	"context"
)

// Context keys
type contextKey string

const (
	// UserIDKey is the key for the user ID in the context
	UserIDKey contextKey = "userID"
)

// NewContextWithUserID adds a user ID to the context
func NewContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// UserIDFromContext extracts the user ID from the context
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

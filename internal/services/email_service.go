package services

import (
	"fmt"
	"math/rand"
	"net/smtp"
	"strings"
	"time"

	"github.com/satonic/satonic-api/internal/config"
)

// EmailService handles email operations
type EmailService struct {
	cfg config.EmailConfig
}

// NewEmailService creates a new EmailService
func NewEmailService(cfg config.EmailConfig) *EmailService {
	return &EmailService{
		cfg: cfg,
	}
}

// SendVerificationCode sends a verification code to an email address
func (s *EmailService) SendVerificationCode(email, code string) error {
	subject := "Satonic - Email Verification Code"
	body := fmt.Sprintf(`
Dear User,

Your email verification code is: %s

This code will expire in 15 minutes.

Best regards,
Satonic Team
`, code)

	return s.SendEmail(email, subject, body)
}

// SendEmail sends an email
func (s *EmailService) SendEmail(to, subject, body string) error {
	// SMTP server configuration
	smtpHost := s.cfg.SMTPHost
	smtpPort := s.cfg.SMTPPort
	smtpUser := s.cfg.SMTPUser
	smtpPassword := s.cfg.SMTPPassword
	from := s.cfg.FromEmail

	// Message
	message := []byte(fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s\r\n", from, to, subject, body))

	// Authentication
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	// SMTP connection
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	// Send email
	if err := smtp.SendMail(addr, auth, from, []string{to}, message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// GenerateVerificationCode generates a random verification code
func (s *EmailService) GenerateVerificationCode(length int) string {
	if length <= 0 {
		length = 6 // Default length
	}

	// Generate a random string of digits
	const digits = "0123456789"
	result := make([]byte, length)

	for i := range result {
		result[i] = digits[rand.Intn(len(digits))]
	}

	return string(result)
}

// GetVerificationExpiry returns the expiry time for verification codes
func (s *EmailService) GetVerificationExpiry(minutes int) time.Time {
	if minutes <= 0 {
		minutes = 15 // Default expiry time
	}

	return time.Now().Add(time.Duration(minutes) * time.Minute)
}

// IsEmailValid checks if an email address is valid
func (s *EmailService) IsEmailValid(email string) bool {
	// Basic validation - check for @ symbol and at least one dot after it
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	// Check if domain has at least one dot
	domainParts := strings.Split(parts[1], ".")
	return len(domainParts) >= 2 && domainParts[len(domainParts)-1] != ""
}

package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/satonic/satonic-api/internal/config"
	"github.com/satonic/satonic-api/internal/models"
	"github.com/satonic/satonic-api/internal/store"
)

// Claims represents the JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthService handles authentication operations
type AuthService struct {
	userRepo      *store.UserRepository
	emailService  *EmailService
	walletService *WalletService
	cfg           config.AuthConfig
}

// NewAuthService creates a new AuthService
func NewAuthService(userRepo *store.UserRepository, emailService *EmailService, walletService *WalletService, cfg config.AuthConfig) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		emailService:  emailService,
		walletService: walletService,
		cfg:           cfg,
	}
}

// AuthenticateWithWallet authenticates a user with a wallet signature
func (s *AuthService) AuthenticateWithWallet(req models.WalletAuthRequest) (*models.AuthToken, error) {
	// Verify the signature
	valid, err := s.walletService.VerifySignature(req.Address, req.Message, req.Signature)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	if !valid {
		return nil, fmt.Errorf("invalid signature")
	}

	// Find or create user based on wallet address
	user, err := s.userRepo.GetByWalletAddress(req.Address)
	if err != nil {
		return nil, err
	}

	// If user doesn't exist, create a new one with this wallet
	if user == nil {
		user, err = s.userRepo.Create()
		if err != nil {
			return nil, err
		}

		// Add the wallet to the user
		_, err = s.userRepo.AddWallet(user.ID, req.Address, "bitcoin")
		if err != nil {
			return nil, err
		}

		// Reload the user to get the wallet
		user, err = s.userRepo.GetByID(user.ID)
		if err != nil {
			return nil, err
		}
	}

	// Generate a JWT token
	token, expiresAt, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.AuthToken{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// AuthenticateWithEmail starts the email authentication process
func (s *AuthService) AuthenticateWithEmail(req models.EmailAuthRequest) error {
	// Validate email
	if !s.emailService.IsEmailValid(req.Email) {
		return fmt.Errorf("invalid email address")
	}

	// Find user with this email
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return err
	}

	// Get email record
	var email *models.Email
	if user != nil {
		// Find the specific email record
		for _, e := range user.Emails {
			if e.Address == req.Email {
				email = &e
				break
			}
		}
	} else {
		// Create a new user
		user, err = s.userRepo.Create()
		if err != nil {
			return err
		}

		// Add the email to the user
		email, err = s.userRepo.AddEmail(user.ID, req.Email, true)
		if err != nil {
			return err
		}
	}

	// Generate verification code
	code := s.emailService.GenerateVerificationCode(s.cfg.CodeLength)

	// Set expiry
	expiresAt := s.emailService.GetVerificationExpiry(s.cfg.CodeExpiration)

	// Store the code
	err = s.userRepo.CreateVerificationCode(email.ID, code, expiresAt)
	if err != nil {
		return err
	}

	// Send the code via email
	return s.emailService.SendVerificationCode(req.Email, code)
}

// VerifyEmailCode verifies an email verification code
func (s *AuthService) VerifyEmailCode(req models.EmailVerifyRequest) (*models.AuthToken, error) {
	// Find user with this email
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("email not found")
	}

	// Find the specific email record
	var email *models.Email
	for _, e := range user.Emails {
		if e.Address == req.Email {
			email = &e
			break
		}
	}

	if email == nil {
		return nil, fmt.Errorf("email not found")
	}

	// Get the latest verification code
	verification, err := s.userRepo.GetVerificationCode(email.ID)
	if err != nil {
		return nil, err
	}

	if verification == nil {
		return nil, fmt.Errorf("no verification code found")
	}

	// Check if code is expired
	if time.Now().After(verification.ExpiresAt) {
		return nil, fmt.Errorf("verification code expired")
	}

	// Check if code matches
	if verification.Code != req.Code {
		return nil, fmt.Errorf("invalid verification code")
	}

	// Mark email as verified
	if !email.Verified {
		err = s.userRepo.VerifyEmail(email.ID)
		if err != nil {
			return nil, err
		}
	}

	// Generate a JWT token
	token, expiresAt, err := s.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Reload the user to get the updated email status
	user, err = s.userRepo.GetByID(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.AuthToken{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

// LinkWallet links a wallet to an existing user
func (s *AuthService) LinkWallet(userID string, req models.WalletAuthRequest) error {
	// Verify the signature
	valid, err := s.walletService.VerifySignature(req.Address, req.Message, req.Signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	if !valid {
		return fmt.Errorf("invalid signature")
	}

	// Check if wallet already exists
	existingWallet, err := s.userRepo.GetWalletByAddress(req.Address)
	if err != nil {
		return err
	}

	if existingWallet != nil && existingWallet.UserID != userID {
		return fmt.Errorf("wallet already linked to another user")
	}

	// Add the wallet to the user
	_, err = s.userRepo.AddWallet(userID, req.Address, "bitcoin")
	return err
}

// LinkEmail links an email to an existing user
func (s *AuthService) LinkEmail(userID string, req models.EmailAuthRequest) error {
	// Validate email
	if !s.emailService.IsEmailValid(req.Email) {
		return fmt.Errorf("invalid email address")
	}

	// Check if email already exists
	existingEmail, err := s.userRepo.GetEmailByAddress(req.Email)
	if err != nil {
		return err
	}

	if existingEmail != nil && existingEmail.UserID != userID {
		return fmt.Errorf("email already linked to another user")
	}

	// Add the email to the user
	email, err := s.userRepo.AddEmail(userID, req.Email, false)
	if err != nil {
		return err
	}

	// Generate verification code
	code := s.emailService.GenerateVerificationCode(s.cfg.CodeLength)

	// Set expiry
	expiresAt := s.emailService.GetVerificationExpiry(s.cfg.CodeExpiration)

	// Store the code
	err = s.userRepo.CreateVerificationCode(email.ID, code, expiresAt)
	if err != nil {
		return err
	}

	// Send the code via email
	return s.emailService.SendVerificationCode(req.Email, code)
}

// ValidateToken validates a JWT token
func (s *AuthService) ValidateToken(tokenString string) (string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	return claims.UserID, nil
}

// generateToken generates a JWT token for a user
func (s *AuthService) generateToken(userID string) (string, time.Time, error) {
	// Set expiration time
	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTExpiration) * time.Hour)

	// Create claims
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "satonic-api",
			Subject:   userID,
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret key
	tokenString, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

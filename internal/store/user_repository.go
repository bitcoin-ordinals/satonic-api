package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/satonic/satonic-api/internal/models"
)

// UserRepository handles database operations related to users
type UserRepository struct {
	db *Database
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, created_at, updated_at FROM users WHERE id = $1`

	err := r.db.GetDB().Get(user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Load wallets
	wallets, err := r.GetWalletsByUserID(id)
	if err != nil {
		return nil, err
	}
	user.Wallets = wallets

	// Load emails
	emails, err := r.GetEmailsByUserID(id)
	if err != nil {
		return nil, err
	}
	user.Emails = emails

	return user, nil
}

// GetByWalletAddress retrieves a user by wallet address
func (r *UserRepository) GetByWalletAddress(address string) (*models.User, error) {
	var userID string
	query := `SELECT user_id FROM wallets WHERE address = $1`

	err := r.db.GetDB().Get(&userID, query, address)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return r.GetByID(userID)
}

// GetByEmail retrieves a user by email address
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var userID string
	query := `SELECT user_id FROM emails WHERE address = $1`

	err := r.db.GetDB().Get(&userID, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return r.GetByID(userID)
}

// Create creates a new user
func (r *UserRepository) Create() (*models.User, error) {
	id := uuid.New().String()
	now := time.Now()

	user := &models.User{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
	}

	query := `INSERT INTO users (id, created_at, updated_at) VALUES ($1, $2, $3)`
	_, err := r.db.GetDB().Exec(query, user.ID, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetWalletsByUserID retrieves wallets for a user
func (r *UserRepository) GetWalletsByUserID(userID string) ([]models.Wallet, error) {
	wallets := []models.Wallet{}
	query := `SELECT id, user_id, address, type, created_at, updated_at 
			  FROM wallets 
			  WHERE user_id = $1`

	err := r.db.GetDB().Select(&wallets, query, userID)
	if err != nil {
		return nil, err
	}

	return wallets, nil
}

// GetWalletByAddress retrieves a wallet by address
func (r *UserRepository) GetWalletByAddress(address string) (*models.Wallet, error) {
	wallet := &models.Wallet{}
	query := `SELECT id, user_id, address, type, created_at, updated_at 
			  FROM wallets 
			  WHERE address = $1`

	err := r.db.GetDB().Get(wallet, query, address)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return wallet, nil
}

// AddWallet adds a wallet to a user
func (r *UserRepository) AddWallet(userID, address, walletType string) (*models.Wallet, error) {
	// Check if wallet already exists
	existingWallet, err := r.GetWalletByAddress(address)
	if err != nil {
		return nil, err
	}

	if existingWallet != nil {
		if existingWallet.UserID != userID {
			return nil, fmt.Errorf("wallet already linked to another user")
		}
		return existingWallet, nil
	}

	id := uuid.New().String()
	now := time.Now()

	wallet := &models.Wallet{
		ID:        id,
		UserID:    userID,
		Address:   address,
		Type:      walletType,
		CreatedAt: now,
		UpdatedAt: now,
	}

	query := `INSERT INTO wallets (id, user_id, address, type, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = r.db.GetDB().Exec(query, wallet.ID, wallet.UserID, wallet.Address, wallet.Type,
		wallet.CreatedAt, wallet.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

// GetEmailsByUserID retrieves emails for a user
func (r *UserRepository) GetEmailsByUserID(userID string) ([]models.Email, error) {
	emails := []models.Email{}
	query := `SELECT id, user_id, address, verified, primary, created_at, updated_at 
			  FROM emails 
			  WHERE user_id = $1`

	err := r.db.GetDB().Select(&emails, query, userID)
	if err != nil {
		return nil, err
	}

	return emails, nil
}

// GetEmailByAddress retrieves an email by address
func (r *UserRepository) GetEmailByAddress(address string) (*models.Email, error) {
	email := &models.Email{}
	query := `SELECT id, user_id, address, verified, primary, created_at, updated_at 
			  FROM emails 
			  WHERE address = $1`

	err := r.db.GetDB().Get(email, query, address)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return email, nil
}

// AddEmail adds an email to a user
func (r *UserRepository) AddEmail(userID, address string, primary bool) (*models.Email, error) {
	// Check if email already exists
	existingEmail, err := r.GetEmailByAddress(address)
	if err != nil {
		return nil, err
	}

	if existingEmail != nil {
		if existingEmail.UserID != userID {
			return nil, fmt.Errorf("email already linked to another user")
		}
		return existingEmail, nil
	}

	// Begin transaction
	return r.AddEmailTx(nil, userID, address, primary)
}

// AddEmailTx adds an email to a user within a transaction
func (r *UserRepository) AddEmailTx(tx *sqlx.Tx, userID, address string, primary bool) (*models.Email, error) {
	var db sqlx.Execer
	if tx != nil {
		db = tx
	} else {
		db = r.db.GetDB()
	}

	// If primary is true, set all other emails to non-primary
	if primary {
		query := `UPDATE emails SET primary = false WHERE user_id = $1`
		_, err := db.Exec(query, userID)
		if err != nil {
			return nil, err
		}
	}

	id := uuid.New().String()
	now := time.Now()

	email := &models.Email{
		ID:        id,
		UserID:    userID,
		Address:   address,
		Verified:  false,
		Primary:   primary,
		CreatedAt: now,
		UpdatedAt: now,
	}

	query := `INSERT INTO emails (id, user_id, address, verified, primary, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.Exec(query, email.ID, email.UserID, email.Address, email.Verified,
		email.Primary, email.CreatedAt, email.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return email, nil
}

// CreateVerificationCode creates an email verification code
func (r *UserRepository) CreateVerificationCode(emailID, code string, expiresAt time.Time) error {
	id := uuid.New().String()
	now := time.Now()

	query := `INSERT INTO email_verifications (id, email_id, code, expires_at, created_at) 
			  VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.GetDB().Exec(query, id, emailID, code, expiresAt, now)
	if err != nil {
		return err
	}

	return nil
}

// VerifyEmail marks an email as verified
func (r *UserRepository) VerifyEmail(emailID string) error {
	query := `UPDATE emails SET verified = true, updated_at = $1 WHERE id = $2`
	_, err := r.db.GetDB().Exec(query, time.Now(), emailID)
	if err != nil {
		return err
	}

	return nil
}

// GetVerificationCode retrieves the latest verification code for an email
func (r *UserRepository) GetVerificationCode(emailID string) (*models.EmailVerification, error) {
	verification := &models.EmailVerification{}
	query := `SELECT id, email_id, code, expires_at, created_at 
			  FROM email_verifications 
			  WHERE email_id = $1 
			  ORDER BY created_at DESC 
			  LIMIT 1`

	err := r.db.GetDB().Get(verification, query, emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return verification, nil
}

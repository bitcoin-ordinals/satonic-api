package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/satonic/satonic-api/internal/models"
)

// NFTRepository handles database operations related to NFTs
type NFTRepository struct {
	db *Database
}

// NewNFTRepository creates a new NFTRepository
func NewNFTRepository(db *Database) *NFTRepository {
	return &NFTRepository{
		db: db,
	}
}

// GetByID retrieves an NFT by ID
func (r *NFTRepository) GetByID(id string) (*models.NFT, error) {
	nft := &models.NFT{}
	query := `SELECT id, wallet_id, token_id, inscription_id, collection, title, 
			  description, image_url, content_url, metadata, created_at, updated_at, auction_id
			  FROM nfts WHERE id = $1`

	err := r.db.GetDB().Get(nft, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return nft, nil
}

// GetByWalletID retrieves NFTs by wallet ID
func (r *NFTRepository) GetByWalletID(walletID string, params models.NFTParams) ([]models.NFT, int, error) {
	nfts := []models.NFT{}

	// Default pagination values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	// Base query
	baseQuery := `FROM nfts WHERE wallet_id = $1`
	args := []interface{}{walletID}
	argCount := 2

	// Add auction filter if provided
	if params.OnAuction != nil {
		if *params.OnAuction {
			baseQuery += ` AND auction_id IS NOT NULL`
		} else {
			baseQuery += ` AND auction_id IS NULL`
		}
	}

	// Add collection filter if provided
	if params.Collection != "" {
		baseQuery += ` AND collection = $` + string('0'+argCount)
		args = append(args, params.Collection)
		argCount++
	}

	// Count total matching records
	var total int
	countQuery := `SELECT COUNT(*) ` + baseQuery
	err := r.db.GetDB().Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (params.Page - 1) * params.PageSize
	selectQuery := `SELECT id, wallet_id, token_id, inscription_id, collection, title, 
				   description, image_url, content_url, metadata, created_at, updated_at, auction_id ` +
		baseQuery + ` ORDER BY created_at DESC LIMIT $` + string('0'+argCount) +
		` OFFSET $` + string('0'+argCount+1)
	args = append(args, params.PageSize, offset)

	err = r.db.GetDB().Select(&nfts, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return nfts, total, nil
}

// GetByUserID retrieves NFTs by user ID
func (r *NFTRepository) GetByUserID(userID string, params models.NFTParams) ([]models.NFT, int, error) {
	nfts := []models.NFT{}

	// Default pagination values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	// Base query joins with wallets to get user's NFTs
	baseQuery := `FROM nfts n 
				 JOIN wallets w ON n.wallet_id = w.id
				 WHERE w.user_id = $1`
	args := []interface{}{userID}
	argCount := 2

	// Add auction filter if provided
	if params.OnAuction != nil {
		if *params.OnAuction {
			baseQuery += ` AND n.auction_id IS NOT NULL`
		} else {
			baseQuery += ` AND n.auction_id IS NULL`
		}
	}

	// Add collection filter if provided
	if params.Collection != "" {
		baseQuery += ` AND n.collection = $` + string('0'+argCount)
		args = append(args, params.Collection)
		argCount++
	}

	// Count total matching records
	var total int
	countQuery := `SELECT COUNT(*) ` + baseQuery
	err := r.db.GetDB().Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (params.Page - 1) * params.PageSize
	selectQuery := `SELECT n.id, n.wallet_id, n.token_id, n.inscription_id, n.collection, n.title, 
				   n.description, n.image_url, n.content_url, n.metadata, n.created_at, n.updated_at, n.auction_id ` +
		baseQuery + ` ORDER BY n.created_at DESC LIMIT $` + string('0'+argCount) +
		` OFFSET $` + string('0'+argCount+1)
	args = append(args, params.PageSize, offset)

	err = r.db.GetDB().Select(&nfts, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return nfts, total, nil
}

// Create creates a new NFT
func (r *NFTRepository) Create(nft *models.NFT) error {
	if nft.ID == "" {
		nft.ID = uuid.New().String()
	}
	now := time.Now()
	nft.CreatedAt = now
	nft.UpdatedAt = now

	query := `INSERT INTO nfts (id, wallet_id, token_id, inscription_id, collection, title, 
			  description, image_url, content_url, metadata, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.db.GetDB().Exec(query,
		nft.ID, nft.WalletID, nft.TokenID, nft.InscriptionID, nft.Collection,
		nft.Title, nft.Description, nft.ImageURL, nft.ContentURL,
		nft.Metadata, nft.CreatedAt, nft.UpdatedAt)

	return err
}

// Update updates an NFT
func (r *NFTRepository) Update(nft *models.NFT) error {
	nft.UpdatedAt = time.Now()

	query := `UPDATE nfts SET wallet_id = $1, token_id = $2, inscription_id = $3, 
			  collection = $4, title = $5, description = $6, image_url = $7, 
			  content_url = $8, metadata = $9, updated_at = $10, auction_id = $11
			  WHERE id = $12`

	_, err := r.db.GetDB().Exec(query,
		nft.WalletID, nft.TokenID, nft.InscriptionID, nft.Collection,
		nft.Title, nft.Description, nft.ImageURL, nft.ContentURL,
		nft.Metadata, nft.UpdatedAt, nft.AuctionID, nft.ID)

	return err
}

// UpdateAuctionID updates the auction ID for an NFT
func (r *NFTRepository) UpdateAuctionID(nftID string, auctionID *string) error {
	query := `UPDATE nfts SET auction_id = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.GetDB().Exec(query, auctionID, time.Now(), nftID)
	return err
}

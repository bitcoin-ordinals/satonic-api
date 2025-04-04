package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/satonic/satonic-api/internal/models"
)

// AuctionRepository handles database operations related to auctions
type AuctionRepository struct {
	db *Database
}

// NewAuctionRepository creates a new AuctionRepository
func NewAuctionRepository(db *Database) *AuctionRepository {
	return &AuctionRepository{
		db: db,
	}
}

// GetByID retrieves an auction by ID
func (r *AuctionRepository) GetByID(id string) (*models.Auction, error) {
	auction := &models.Auction{}
	query := `SELECT id, nft_id, seller_wallet_id, start_price, reserve_price, buy_now_price, 
			  current_bid, current_bidder_id, start_time, end_time, status, psbt, 
			  created_at, updated_at
			  FROM auctions WHERE id = $1`

	err := r.db.GetDB().Get(auction, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return auction, nil
}

// GetByIDWithNFT retrieves an auction by ID with its associated NFT
func (r *AuctionRepository) GetByIDWithNFT(id string) (*models.Auction, error) {
	auction, err := r.GetByID(id)
	if err != nil || auction == nil {
		return nil, err
	}

	// Fetch associated NFT
	query := `SELECT id, wallet_id, token_id, inscription_id, collection, title, 
			  description, image_url, content_url, metadata, created_at, updated_at, auction_id
			  FROM nfts WHERE id = $1`

	nft := &models.NFT{}
	err = r.db.GetDB().Get(nft, query, auction.NFTID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	auction.NFT = nft

	// Fetch bids
	bids, err := r.GetBidsByAuctionID(id)
	if err != nil {
		return nil, err
	}

	auction.Bids = bids

	return auction, nil
}

// List retrieves auctions based on filter parameters
func (r *AuctionRepository) List(params models.AuctionParams) ([]models.Auction, int, error) {
	auctions := []models.Auction{}

	// Default pagination values
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	}

	// Base query
	baseQuery := `FROM auctions a`
	whereClause := ``
	args := []interface{}{}
	argCount := 1

	// Add status filter if provided
	if params.Status != "" {
		if whereClause == "" {
			whereClause = ` WHERE`
		} else {
			whereClause += ` AND`
		}
		whereClause += ` a.status = $` + string('0'+argCount)
		args = append(args, params.Status)
		argCount++
	}

	// Add seller filter if provided
	if params.SellerID != "" {
		if whereClause == "" {
			whereClause = ` WHERE`
		} else {
			whereClause += ` AND`
		}
		// Join with wallets to filter by seller user ID
		baseQuery += ` JOIN wallets w ON a.seller_wallet_id = w.id`
		whereClause += ` w.user_id = $` + string('0'+argCount)
		args = append(args, params.SellerID)
		argCount++
	}

	// Add bidder filter if provided
	if params.BidderID != "" {
		if whereClause == "" {
			whereClause = ` WHERE`
		} else {
			whereClause += ` AND`
		}
		// Subquery to find auctions where user has placed bids
		whereClause += ` a.id IN (SELECT auction_id FROM bids b 
								 JOIN wallets w ON b.wallet_id = w.id 
								 WHERE w.user_id = $` + string('0'+argCount) + `)`
		args = append(args, params.BidderID)
		argCount++
	}

	// Complete the query
	baseQuery += whereClause

	// Count total matching records
	var total int
	countQuery := `SELECT COUNT(*) ` + baseQuery
	err := r.db.GetDB().Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (params.Page - 1) * params.PageSize
	selectQuery := `SELECT a.id, a.nft_id, a.seller_wallet_id, a.start_price, a.reserve_price, 
				   a.buy_now_price, a.current_bid, a.current_bidder_id, a.start_time, a.end_time, 
				   a.status, a.psbt, a.created_at, a.updated_at ` +
		baseQuery + ` ORDER BY a.end_time ASC LIMIT $` + string('0'+argCount) +
		` OFFSET $` + string('0'+argCount+1)
	args = append(args, params.PageSize, offset)

	err = r.db.GetDB().Select(&auctions, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Load NFTs and bids for each auction
	for i := range auctions {
		// Fetch associated NFT
		query := `SELECT id, wallet_id, token_id, inscription_id, collection, title, 
				 description, image_url, content_url, metadata, created_at, updated_at, auction_id
				 FROM nfts WHERE id = $1`

		nft := &models.NFT{}
		err = r.db.GetDB().Get(nft, query, auctions[i].NFTID)
		if err != nil && err != sql.ErrNoRows {
			continue
		}

		auctions[i].NFT = nft

		// Fetch top 3 bids
		bids, err := r.GetTopBidsByAuctionID(auctions[i].ID, 3)
		if err != nil {
			continue
		}

		auctions[i].Bids = bids
	}

	return auctions, total, nil
}

// Create creates a new auction
func (r *AuctionRepository) Create(auction *models.Auction) error {
	// Use transaction to ensure NFT is properly linked to auction
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		if auction.ID == "" {
			auction.ID = uuid.New().String()
		}
		now := time.Now()
		auction.CreatedAt = now
		auction.UpdatedAt = now

		// Set initial status
		if auction.Status == "" {
			if now.After(auction.StartTime) {
				auction.Status = models.AuctionStatusActive
			} else {
				auction.Status = models.AuctionStatusDraft
			}
		}

		// Insert auction
		query := `INSERT INTO auctions (id, nft_id, seller_wallet_id, start_price, reserve_price, 
				 buy_now_price, start_time, end_time, status, psbt, created_at, updated_at) 
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

		_, err := tx.Exec(query,
			auction.ID, auction.NFTID, auction.SellerWalletID, auction.StartPrice,
			auction.ReservePrice, auction.BuyNowPrice, auction.StartTime,
			auction.EndTime, auction.Status, auction.PSBT, auction.CreatedAt, auction.UpdatedAt)

		if err != nil {
			return err
		}

		// Update NFT with auction ID
		query = `UPDATE nfts SET auction_id = $1, updated_at = $2 WHERE id = $3`
		_, err = tx.Exec(query, auction.ID, now, auction.NFTID)
		if err != nil {
			return err
		}

		return nil
	})
}

// Update updates an auction
func (r *AuctionRepository) Update(auction *models.Auction) error {
	auction.UpdatedAt = time.Now()

	query := `UPDATE auctions SET nft_id = $1, seller_wallet_id = $2, start_price = $3, 
			 reserve_price = $4, buy_now_price = $5, current_bid = $6, current_bidder_id = $7,
			 start_time = $8, end_time = $9, status = $10, psbt = $11, updated_at = $12
			 WHERE id = $13`

	_, err := r.db.GetDB().Exec(query,
		auction.NFTID, auction.SellerWalletID, auction.StartPrice,
		auction.ReservePrice, auction.BuyNowPrice, auction.CurrentBid,
		auction.CurrentBidderID, auction.StartTime, auction.EndTime,
		auction.Status, auction.PSBT, auction.UpdatedAt, auction.ID)

	return err
}

// UpdateStatus updates the status of an auction
func (r *AuctionRepository) UpdateStatus(id string, status models.AuctionStatus) error {
	query := `UPDATE auctions SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.GetDB().Exec(query, status, time.Now(), id)
	return err
}

// CompleteAuction completes an auction and releases the NFT
func (r *AuctionRepository) CompleteAuction(auctionID string, status models.AuctionStatus) error {
	// Use transaction to ensure NFT is properly updated
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		now := time.Now()

		// Update auction status
		query := `UPDATE auctions SET status = $1, updated_at = $2 WHERE id = $3`
		_, err := tx.Exec(query, status, now, auctionID)
		if err != nil {
			return err
		}

		if status == models.AuctionStatusCompleted {
			// If completed, keep the auction_id (for history)
			return nil
		} else {
			// If cancelled, remove the auction_id from NFT
			query = `UPDATE nfts SET auction_id = NULL, updated_at = $1 
					WHERE auction_id = $2`
			_, err = tx.Exec(query, now, auctionID)
			return err
		}
	})
}

// CreateBid creates a new bid
func (r *AuctionRepository) CreateBid(bid *models.Bid) error {
	// Use transaction to update auction if bid is higher than current
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		if bid.ID == "" {
			bid.ID = uuid.New().String()
		}
		now := time.Now()
		bid.CreatedAt = now

		// Insert bid
		query := `INSERT INTO bids (id, auction_id, bidder_id, wallet_id, amount, created_at, accepted) 
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`

		_, err := tx.Exec(query,
			bid.ID, bid.AuctionID, bid.BidderID, bid.WalletID,
			bid.Amount, bid.CreatedAt, bid.Accepted)

		if err != nil {
			return err
		}

		// Check if this is the highest bid
		var currentBid sql.NullInt64
		query = `SELECT current_bid FROM auctions WHERE id = $1`
		err = tx.Get(&currentBid, query, bid.AuctionID)
		if err != nil {
			return err
		}

		if !currentBid.Valid || bid.Amount > currentBid.Int64 {
			// Update auction with new highest bid
			query = `UPDATE auctions SET current_bid = $1, current_bidder_id = $2, updated_at = $3 
					WHERE id = $4`
			_, err = tx.Exec(query, bid.Amount, bid.BidderID, now, bid.AuctionID)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// GetBidsByAuctionID retrieves bids for an auction
func (r *AuctionRepository) GetBidsByAuctionID(auctionID string) ([]models.Bid, error) {
	bids := []models.Bid{}
	query := `SELECT id, auction_id, bidder_id, wallet_id, amount, created_at, accepted, signature 
			 FROM bids 
			 WHERE auction_id = $1 
			 ORDER BY amount DESC`

	err := r.db.GetDB().Select(&bids, query, auctionID)
	if err != nil {
		return nil, err
	}

	return bids, nil
}

// GetTopBidsByAuctionID retrieves top N bids for an auction
func (r *AuctionRepository) GetTopBidsByAuctionID(auctionID string, limit int) ([]models.Bid, error) {
	bids := []models.Bid{}
	query := `SELECT id, auction_id, bidder_id, wallet_id, amount, created_at, accepted, signature 
			 FROM bids 
			 WHERE auction_id = $1 
			 ORDER BY amount DESC
			 LIMIT $2`

	err := r.db.GetDB().Select(&bids, query, auctionID, limit)
	if err != nil {
		return nil, err
	}

	return bids, nil
}

// GetActiveAuctions retrieves all active auctions
func (r *AuctionRepository) GetActiveAuctions() ([]models.Auction, error) {
	auctions := []models.Auction{}
	query := `SELECT id, nft_id, seller_wallet_id, start_price, reserve_price, buy_now_price, 
			 current_bid, current_bidder_id, start_time, end_time, status, psbt, created_at, updated_at
			 FROM auctions 
			 WHERE status = $1 AND end_time > $2
			 ORDER BY end_time ASC`

	err := r.db.GetDB().Select(&auctions, query, models.AuctionStatusActive, time.Now())
	if err != nil {
		return nil, err
	}

	return auctions, nil
}

// GetEndedAuctions retrieves auctions that have ended but not yet finalized
func (r *AuctionRepository) GetEndedAuctions() ([]models.Auction, error) {
	auctions := []models.Auction{}
	query := `SELECT id, nft_id, seller_wallet_id, start_price, reserve_price, buy_now_price, 
			 current_bid, current_bidder_id, start_time, end_time, status, psbt, created_at, updated_at
			 FROM auctions 
			 WHERE status = $1 AND end_time <= $2
			 ORDER BY end_time ASC`

	err := r.db.GetDB().Select(&auctions, query, models.AuctionStatusActive, time.Now())
	if err != nil {
		return nil, err
	}

	return auctions, nil
}

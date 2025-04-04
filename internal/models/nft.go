package models

import (
	"encoding/json"
	"time"
)

// NFT represents an NFT (Non-Fungible Token) in the system
type NFT struct {
	ID          string          `json:"id" db:"id"`
	WalletID    string          `json:"wallet_id" db:"wallet_id"`
	TokenID     string          `json:"token_id" db:"token_id"`
	InscriptionID string        `json:"inscription_id" db:"inscription_id"`
	Collection  string          `json:"collection" db:"collection"`
	Title       string          `json:"title" db:"title"`
	Description string          `json:"description" db:"description"`
	ImageURL    string          `json:"image_url" db:"image_url"`
	ContentURL  string          `json:"content_url" db:"content_url"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	AuctionID   *string         `json:"auction_id,omitempty" db:"auction_id"`
}

// NFTListResponse represents the response for listing NFTs
type NFTListResponse struct {
	NFTs       []NFT  `json:"nfts"`
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// NFTParams represents the parameters for filtering NFTs
type NFTParams struct {
	WalletID   string `json:"wallet_id"`
	UserID     string `json:"user_id"`
	Collection string `json:"collection"`
	OnAuction  *bool  `json:"on_auction"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
} 
package models

import (
	"time"
)

// AuctionStatus represents the status of an auction
type AuctionStatus string

const (
	AuctionStatusDraft     AuctionStatus = "draft"
	AuctionStatusActive    AuctionStatus = "active"
	AuctionStatusCompleted AuctionStatus = "completed"
	AuctionStatusCancelled AuctionStatus = "cancelled"
)

// Auction represents an NFT auction in the system
type Auction struct {
	ID            string        `json:"id" db:"id"`
	NFTID         string        `json:"nft_id" db:"nft_id"`
	SellerWalletID string       `json:"seller_wallet_id" db:"seller_wallet_id"`
	StartPrice    int64         `json:"start_price" db:"start_price"` // in satoshis
	ReservePrice  *int64        `json:"reserve_price,omitempty" db:"reserve_price"`
	BuyNowPrice   *int64        `json:"buy_now_price,omitempty" db:"buy_now_price"`
	CurrentBid    *int64        `json:"current_bid,omitempty" db:"current_bid"`
	CurrentBidderID *string     `json:"current_bidder_id,omitempty" db:"current_bidder_id"`
	StartTime     time.Time     `json:"start_time" db:"start_time"`
	EndTime       time.Time     `json:"end_time" db:"end_time"`
	Status        AuctionStatus `json:"status" db:"status"`
	PSBT          string        `json:"psbt" db:"psbt"` // Partially Signed Bitcoin Transaction
	CreatedAt     time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at" db:"updated_at"`
	NFT           *NFT          `json:"nft,omitempty"`
	Bids          []Bid         `json:"bids,omitempty"`
}

// Bid represents a bid on an auction
type Bid struct {
	ID         string    `json:"id" db:"id"`
	AuctionID  string    `json:"auction_id" db:"auction_id"`
	BidderID   string    `json:"bidder_id" db:"bidder_id"`
	WalletID   string    `json:"wallet_id" db:"wallet_id"`
	Amount     int64     `json:"amount" db:"amount"` // in satoshis
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	Accepted   bool      `json:"accepted" db:"accepted"`
	Signature  *string   `json:"signature,omitempty" db:"signature"`
}

// CreateAuctionRequest represents a request to create an auction
type CreateAuctionRequest struct {
	NFTID        string    `json:"nft_id"`
	StartPrice   int64     `json:"start_price"`
	ReservePrice *int64    `json:"reserve_price,omitempty"`
	BuyNowPrice  *int64    `json:"buy_now_price,omitempty"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	PSBT         string    `json:"psbt"`
}

// PlaceBidRequest represents a request to place a bid on an auction
type PlaceBidRequest struct {
	AuctionID string `json:"auction_id"`
	Amount    int64  `json:"amount"`
	WalletID  string `json:"wallet_id"`
}

// FinalizeAuctionRequest represents a request to finalize an auction
type FinalizeAuctionRequest struct {
	AuctionID  string `json:"auction_id"`
	Signature  string `json:"signature"`
}

// AuctionListResponse represents the response for listing auctions
type AuctionListResponse struct {
	Auctions   []Auction `json:"auctions"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// AuctionParams represents the parameters for filtering auctions
type AuctionParams struct {
	Status     AuctionStatus `json:"status"`
	SellerID   string        `json:"seller_id"`
	BidderID   string        `json:"bidder_id"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
} 
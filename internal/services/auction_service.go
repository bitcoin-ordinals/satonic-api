package services

import (
	"fmt"
	"time"

	"github.com/satonic/satonic-api/internal/models"
	"github.com/satonic/satonic-api/internal/store"
)

// AuctionService handles auction operations
type AuctionService struct {
	auctionRepo *store.AuctionRepository
	nftRepo     *store.NFTRepository
	userRepo    *store.UserRepository
}

// NewAuctionService creates a new AuctionService
func NewAuctionService(auctionRepo *store.AuctionRepository, nftRepo *store.NFTRepository, userRepo *store.UserRepository) *AuctionService {
	return &AuctionService{
		auctionRepo: auctionRepo,
		nftRepo:     nftRepo,
		userRepo:    userRepo,
	}
}

// GetByID retrieves an auction by ID
func (s *AuctionService) GetByID(id string) (*models.Auction, error) {
	return s.auctionRepo.GetByIDWithNFT(id)
}

// List retrieves auctions based on filter parameters
func (s *AuctionService) List(params models.AuctionParams) (*models.AuctionListResponse, error) {
	auctions, total, err := s.auctionRepo.List(params)
	if err != nil {
		return nil, err
	}

	return &models.AuctionListResponse{
		Auctions:   auctions,
		TotalCount: total,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}, nil
}

// Create creates a new auction
func (s *AuctionService) Create(req models.CreateAuctionRequest, userID string) (*models.Auction, error) {
	// Check if NFT exists and belongs to the user
	nft, err := s.nftRepo.GetByID(req.NFTID)
	if err != nil {
		return nil, err
	}

	if nft == nil {
		return nil, fmt.Errorf("NFT not found")
	}

	// Check if the NFT is already on auction
	if nft.AuctionID != nil {
		return nil, fmt.Errorf("NFT is already on auction")
	}

	// Get the wallet
	wallet, err := s.userRepo.GetWalletsByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Find the wallet that owns the NFT
	var sellerWallet *models.Wallet
	for _, w := range wallet {
		if w.ID == nft.WalletID {
			sellerWallet = &w
			break
		}
	}

	if sellerWallet == nil {
		return nil, fmt.Errorf("NFT is not owned by the user")
	}

	// Validate the PSBT
	walletService := NewWalletService()
	valid, message, err := walletService.ValidatePSBT(req.PSBT, nft.InscriptionID, sellerWallet.Address, "")
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, fmt.Errorf("invalid PSBT: %s", message)
	}

	// Create auction
	auction := &models.Auction{
		NFTID:          req.NFTID,
		SellerWalletID: sellerWallet.ID,
		StartPrice:     req.StartPrice,
		ReservePrice:   req.ReservePrice,
		BuyNowPrice:    req.BuyNowPrice,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Status:         models.AuctionStatusDraft,
		PSBT:           req.PSBT,
	}

	// If start time is in the past or now, set status to active
	if !req.StartTime.After(time.Now()) {
		auction.Status = models.AuctionStatusActive
	}

	// Create the auction
	err = s.auctionRepo.Create(auction)
	if err != nil {
		return nil, err
	}

	// Fetch the full auction with NFT
	return s.GetByID(auction.ID)
}

// PlaceBid places a bid on an auction
func (s *AuctionService) PlaceBid(req models.PlaceBidRequest, userID string) (*models.Bid, error) {
	// Get the auction
	auction, err := s.GetByID(req.AuctionID)
	if err != nil {
		return nil, err
	}

	if auction == nil {
		return nil, fmt.Errorf("auction not found")
	}

	// Check if auction is active
	if auction.Status != models.AuctionStatusActive {
		return nil, fmt.Errorf("auction is not active")
	}

	// Check if auction has started
	if time.Now().Before(auction.StartTime) {
		return nil, fmt.Errorf("auction has not started yet")
	}

	// Check if auction has ended
	if time.Now().After(auction.EndTime) {
		return nil, fmt.Errorf("auction has ended")
	}

	// Check if bid amount is higher than current bid
	if auction.CurrentBid != nil && req.Amount <= *auction.CurrentBid {
		return nil, fmt.Errorf("bid amount must be higher than current bid")
	}

	// Check if bid amount is at least the start price
	if req.Amount < auction.StartPrice {
		return nil, fmt.Errorf("bid amount must be at least the start price")
	}

	// Verify wallet belongs to user
	wallet, err := s.userRepo.GetWalletsByUserID(userID)
	if err != nil {
		return nil, err
	}

	var bidderWallet *models.Wallet
	for _, w := range wallet {
		if w.ID == req.WalletID {
			bidderWallet = &w
			break
		}
	}

	if bidderWallet == nil {
		return nil, fmt.Errorf("wallet not found or not owned by user")
	}

	// Check if bidder has enough balance
	walletService := NewWalletService()
	balance, err := walletService.GetBalance(bidderWallet.Address)
	if err != nil {
		return nil, err
	}

	if balance < req.Amount {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Create bid
	bid := &models.Bid{
		AuctionID: req.AuctionID,
		BidderID:  userID,
		WalletID:  req.WalletID,
		Amount:    req.Amount,
		Accepted:  true,
	}

	// Save bid
	err = s.auctionRepo.CreateBid(bid)
	if err != nil {
		return nil, err
	}

	return bid, nil
}

// FinalizeAuction finalizes an auction
func (s *AuctionService) FinalizeAuction(req models.FinalizeAuctionRequest, userID string) (*models.Auction, error) {
	// Get the auction
	auction, err := s.GetByID(req.AuctionID)
	if err != nil {
		return nil, err
	}

	if auction == nil {
		return nil, fmt.Errorf("auction not found")
	}

	// Check if auction is active
	if auction.Status != models.AuctionStatusActive {
		return nil, fmt.Errorf("auction is not active")
	}

	// Check if auction has ended or has a "Buy Now" price that was met
	buyNowTriggered := auction.BuyNowPrice != nil &&
		auction.CurrentBid != nil &&
		*auction.CurrentBid >= *auction.BuyNowPrice

	if !time.Now().After(auction.EndTime) && !buyNowTriggered {
		return nil, fmt.Errorf("auction has not ended yet")
	}

	// Check if there are any bids
	if auction.CurrentBid == nil || auction.CurrentBidderID == nil {
		// No bids, cancel the auction
		err = s.auctionRepo.CompleteAuction(auction.ID, models.AuctionStatusCancelled)
		if err != nil {
			return nil, err
		}

		auction.Status = models.AuctionStatusCancelled
		return auction, nil
	}

	// Check if reserve price was met
	if auction.ReservePrice != nil && *auction.CurrentBid < *auction.ReservePrice {
		// Reserve not met, cancel the auction
		err = s.auctionRepo.CompleteAuction(auction.ID, models.AuctionStatusCancelled)
		if err != nil {
			return nil, err
		}

		auction.Status = models.AuctionStatusCancelled
		return auction, nil
	}

	// Get the winning bidder
	if *auction.CurrentBidderID != userID {
		return nil, fmt.Errorf("only the winning bidder can finalize the auction")
	}

	// Validate the signature
	// In a real implementation, this would complete the PSBT transaction
	// by adding the winning bidder's signature

	// Complete the auction
	err = s.auctionRepo.CompleteAuction(auction.ID, models.AuctionStatusCompleted)
	if err != nil {
		return nil, err
	}

	// Update auction status
	auction.Status = models.AuctionStatusCompleted

	return auction, nil
}

// GetActiveAuctions retrieves all active auctions
func (s *AuctionService) GetActiveAuctions() ([]models.Auction, error) {
	return s.auctionRepo.GetActiveAuctions()
}

// ProcessEndedAuctions processes auctions that have ended but not yet finalized
func (s *AuctionService) ProcessEndedAuctions() error {
	// Get all ended auctions
	auctions, err := s.auctionRepo.GetEndedAuctions()
	if err != nil {
		return err
	}

	for _, auction := range auctions {
		// Check if there are any bids
		if auction.CurrentBid == nil || auction.CurrentBidderID == nil {
			// No bids, cancel the auction
			err = s.auctionRepo.CompleteAuction(auction.ID, models.AuctionStatusCancelled)
			if err != nil {
				return err
			}
			continue
		}

		// Check if reserve price was met
		if auction.ReservePrice != nil && *auction.CurrentBid < *auction.ReservePrice {
			// Reserve not met, cancel the auction
			err = s.auctionRepo.CompleteAuction(auction.ID, models.AuctionStatusCancelled)
			if err != nil {
				return err
			}
			continue
		}

		// Auction has a winning bid, but needs to be finalized by the bidder
		// Send notification to the bidder (in a real implementation)
	}

	return nil
}

package services

import (
	"encoding/json"
	"fmt"

	"github.com/satonic/satonic-api/internal/models"
	"github.com/satonic/satonic-api/internal/store"
)

// NFTService handles NFT-related operations
type NFTService struct {
	nftRepo *store.NFTRepository
}

// NewNFTService creates a new NFTService
func NewNFTService(nftRepo *store.NFTRepository) *NFTService {
	return &NFTService{
		nftRepo: nftRepo,
	}
}

// GetByID retrieves an NFT by ID
func (s *NFTService) GetByID(id string) (*models.NFT, error) {
	return s.nftRepo.GetByID(id)
}

// GetByWalletID retrieves NFTs owned by a wallet
func (s *NFTService) GetByWalletID(walletID string, params models.NFTParams) (*models.NFTListResponse, error) {
	nfts, total, err := s.nftRepo.GetByWalletID(walletID, params)
	if err != nil {
		return nil, err
	}

	return &models.NFTListResponse{
		NFTs:       nfts,
		TotalCount: total,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}, nil
}

// GetByUserID retrieves NFTs owned by a user across all their wallets
func (s *NFTService) GetByUserID(userID string, params models.NFTParams) (*models.NFTListResponse, error) {
	nfts, total, err := s.nftRepo.GetByUserID(userID, params)
	if err != nil {
		return nil, err
	}

	return &models.NFTListResponse{
		NFTs:       nfts,
		TotalCount: total,
		Page:       params.Page,
		PageSize:   params.PageSize,
	}, nil
}

// Create creates a new NFT
func (s *NFTService) Create(nft *models.NFT) error {
	return s.nftRepo.Create(nft)
}

// Update updates an NFT
func (s *NFTService) Update(nft *models.NFT) error {
	return s.nftRepo.Update(nft)
}

// ValidateOrdinal validates an ordinal inscription
func (s *NFTService) ValidateOrdinal(inscriptionID, walletAddress string) (bool, error) {
	// In a real implementation, you would:
	// 1. Query a Bitcoin node or API to check the inscription
	// 2. Verify the inscription belongs to the wallet address
	// 3. Parse the metadata to ensure it's a valid NFT

	// This is a placeholder for demo purposes
	return true, nil
}

// ImportOrdinal imports an ordinal as an NFT
func (s *NFTService) ImportOrdinal(walletID, inscriptionID string) (*models.NFT, error) {
	// In a real implementation, you would:
	// 1. Fetch the inscription details from a Bitcoin node or API
	// 2. Parse the metadata to extract NFT information
	// 3. Create a new NFT record

	// This is a placeholder for demo purposes
	nft := &models.NFT{
		WalletID:      walletID,
		InscriptionID: inscriptionID,
		TokenID:       inscriptionID, // Using inscription ID as token ID
		Collection:    "Ordinals",
		Title:         "Ordinal #" + inscriptionID[:8],
		Description:   "An Ordinal inscription",
		ImageURL:      "https://example.com/ordinals/" + inscriptionID + ".png",
		ContentURL:    "https://example.com/ordinals/" + inscriptionID + ".json",
		Metadata:      json.RawMessage(`{"type":"ordinal","rarity":"common"}`),
	}

	// Save the NFT
	err := s.Create(nft)
	if err != nil {
		return nil, fmt.Errorf("failed to import ordinal: %w", err)
	}

	return nft, nil
}

// IsOwnedByUser checks if an NFT is owned by a specific user
func (s *NFTService) IsOwnedByUser(nftID, userID string, userRepo *store.UserRepository) (bool, error) {
	// Get the NFT
	nft, err := s.GetByID(nftID)
	if err != nil {
		return false, err
	}

	if nft == nil {
		return false, fmt.Errorf("NFT not found")
	}

	// Get wallets for the user
	wallets, err := userRepo.GetWalletsByUserID(userID)
	if err != nil {
		return false, err
	}

	// Check if any of the user's wallets owns the NFT
	for _, wallet := range wallets {
		if wallet.ID == nft.WalletID {
			return true, nil
		}
	}

	return false, nil
}

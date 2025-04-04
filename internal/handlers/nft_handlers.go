package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/satonic/satonic-api/internal/models"
	"github.com/satonic/satonic-api/internal/services"
)

// GetUserNFTs handles retrieving a user's NFTs
func GetUserNFTs(nftService *services.NFTService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID := r.Context().Value(UserIDKey).(string)

		// Parse query parameters
		params := parseNFTParams(r)

		// Get NFTs for user
		response, err := nftService.GetByUserID(userID, params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return NFTs
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetNFT handles retrieving a single NFT
func GetNFT(nftService *services.NFTService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get NFT ID from URL
		nftID := chi.URLParam(r, "id")
		if nftID == "" {
			http.Error(w, "NFT ID is required", http.StatusBadRequest)
			return
		}

		// Get NFT
		nft, err := nftService.GetByID(nftID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if nft == nil {
			http.Error(w, "NFT not found", http.StatusNotFound)
			return
		}

		// Return NFT
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nft)
	}
}

// Helper function to parse NFT query parameters
func parseNFTParams(r *http.Request) models.NFTParams {
	params := models.NFTParams{}

	// Get collection filter
	params.Collection = r.URL.Query().Get("collection")

	// Get on_auction filter
	onAuctionStr := r.URL.Query().Get("on_auction")
	if onAuctionStr != "" {
		onAuction := onAuctionStr == "true"
		params.OnAuction = &onAuction
	}

	// Get pagination
	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil && page > 0 {
			params.Page = page
		}
	}

	pageSizeStr := r.URL.Query().Get("page_size")
	if pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSize > 0 {
			params.PageSize = pageSize
		}
	}

	return params
}

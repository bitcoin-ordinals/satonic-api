package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/satonic/satonic-api/internal/models"
	"github.com/satonic/satonic-api/internal/services"
)

// GetAllAuctions handles retrieving all auctions
func GetAllAuctions(auctionService *services.AuctionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		params := parseAuctionParams(r)

		// Get auctions
		response, err := auctionService.List(params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return auctions
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetAuction handles retrieving a single auction
func GetAuction(auctionService *services.AuctionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get auction ID from URL
		auctionID := chi.URLParam(r, "id")
		if auctionID == "" {
			http.Error(w, "Auction ID is required", http.StatusBadRequest)
			return
		}

		// Get auction
		auction, err := auctionService.GetByID(auctionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if auction == nil {
			http.Error(w, "Auction not found", http.StatusNotFound)
			return
		}

		// Return auction
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(auction)
	}
}

// CreateAuction handles creating a new auction
func CreateAuction(auctionService *services.AuctionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID := r.Context().Value(UserIDKey).(string)

		// Parse request body
		var req models.CreateAuctionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create auction
		auction, err := auctionService.Create(req, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return auction
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(auction)
	}
}

// FinalizeAuction handles finalizing an auction
func FinalizeAuction(auctionService *services.AuctionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID := r.Context().Value(UserIDKey).(string)

		// Get auction ID from URL
		auctionID := chi.URLParam(r, "id")
		if auctionID == "" {
			http.Error(w, "Auction ID is required", http.StatusBadRequest)
			return
		}

		// Parse request body
		var req models.FinalizeAuctionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Set auction ID from URL
		req.AuctionID = auctionID

		// Finalize auction
		auction, err := auctionService.FinalizeAuction(req, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return auction
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(auction)
	}
}

// Helper function to parse auction query parameters
func parseAuctionParams(r *http.Request) models.AuctionParams {
	params := models.AuctionParams{}

	// Get status filter
	statusStr := r.URL.Query().Get("status")
	if statusStr != "" {
		params.Status = models.AuctionStatus(statusStr)
	}

	// Get seller filter
	params.SellerID = r.URL.Query().Get("seller_id")

	// Get bidder filter
	params.BidderID = r.URL.Query().Get("bidder_id")

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

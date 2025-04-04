package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/satonic/satonic-api/internal/models"
	"github.com/satonic/satonic-api/internal/services"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins (for development)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// BidMessage represents a bid message sent over WebSocket
type BidMessage struct {
	AuctionID string `json:"auction_id"`
	WalletID  string `json:"wallet_id"`
	Amount    int64  `json:"amount"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	// User details
	userID string
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Clients by auction ID that they're watching
	auctionClients map[string]map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Auction service
	auctionService *services.AuctionService
}

// NewHub creates a new hub
func NewHub(auctionService *services.AuctionService) *Hub {
	return &Hub{
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		clients:        make(map[*Client]bool),
		auctionClients: make(map[string]map[*Client]bool),
		auctionService: auctionService,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			// Broadcast message to all clients
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// RegisterAuctionClient registers a client to receive updates for a specific auction
func (h *Hub) RegisterAuctionClient(client *Client, auctionID string) {
	if _, ok := h.auctionClients[auctionID]; !ok {
		h.auctionClients[auctionID] = make(map[*Client]bool)
	}
	h.auctionClients[auctionID][client] = true
}

// UnregisterAuctionClient unregisters a client from receiving updates for a specific auction
func (h *Hub) UnregisterAuctionClient(client *Client, auctionID string) {
	if _, ok := h.auctionClients[auctionID]; ok {
		delete(h.auctionClients[auctionID], client)
		if len(h.auctionClients[auctionID]) == 0 {
			delete(h.auctionClients, auctionID)
		}
	}
}

// BroadcastToAuction broadcasts a message to all clients subscribed to an auction
func (h *Hub) BroadcastToAuction(auctionID string, message []byte) {
	if clients, ok := h.auctionClients[auctionID]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
				delete(clients, client)
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Parse the message
		var wsMessage WebSocketMessage
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("error parsing message: %v", err)
			continue
		}

		// Handle different message types
		switch wsMessage.Type {
		case "subscribe":
			// Subscribe to auction updates
			var auctionID string
			if err := json.Unmarshal(wsMessage.Payload, &auctionID); err != nil {
				log.Printf("error parsing subscribe payload: %v", err)
				continue
			}
			c.hub.RegisterAuctionClient(c, auctionID)

		case "unsubscribe":
			// Unsubscribe from auction updates
			var auctionID string
			if err := json.Unmarshal(wsMessage.Payload, &auctionID); err != nil {
				log.Printf("error parsing unsubscribe payload: %v", err)
				continue
			}
			c.hub.UnregisterAuctionClient(c, auctionID)

		case "bid":
			// Place a bid
			var bidMessage BidMessage
			if err := json.Unmarshal(wsMessage.Payload, &bidMessage); err != nil {
				log.Printf("error parsing bid payload: %v", err)
				continue
			}

			// Ensure user is authenticated
			if c.userID == "" {
				response := WebSocketMessage{
					Type:    "error",
					Payload: json.RawMessage(`{"message":"Not authenticated"}`),
				}
				responseBytes, _ := json.Marshal(response)
				c.send <- responseBytes
				continue
			}

			// Place the bid
			bidRequest := models.PlaceBidRequest{
				AuctionID: bidMessage.AuctionID,
				Amount:    bidMessage.Amount,
				WalletID:  bidMessage.WalletID,
			}

			bid, err := c.hub.auctionService.PlaceBid(bidRequest, c.userID)
			if err != nil {
				response := WebSocketMessage{
					Type:    "error",
					Payload: json.RawMessage(`{"message":"` + err.Error() + `"}`),
				}
				responseBytes, _ := json.Marshal(response)
				c.send <- responseBytes
				continue
			}

			// Get the updated auction
			auction, err := c.hub.auctionService.GetByID(bidMessage.AuctionID)
			if err != nil {
				log.Printf("error getting auction: %v", err)
				continue
			}

			// Broadcast the updated auction to all subscribers
			auctionBytes, err := json.Marshal(auction)
			if err != nil {
				log.Printf("error marshalling auction: %v", err)
				continue
			}

			response := WebSocketMessage{
				Type:    "auction_update",
				Payload: auctionBytes,
			}
			responseBytes, _ := json.Marshal(response)
			c.hub.BroadcastToAuction(bidMessage.AuctionID, responseBytes)

			// Send a confirmation to the bidder
			bidBytes, err := json.Marshal(bid)
			if err != nil {
				log.Printf("error marshalling bid: %v", err)
				continue
			}

			bidResponse := WebSocketMessage{
				Type:    "bid_placed",
				Payload: bidBytes,
			}
			bidResponseBytes, _ := json.Marshal(bidResponse)
			c.send <- bidResponseBytes
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles WebSocket requests from clients
func ServeWs(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		// Get user ID from token (if available)
		userID := r.Context().Value("userID").(string)

		client := &Client{
			hub:    hub,
			conn:   conn,
			send:   make(chan []byte, 256),
			userID: userID,
		}
		client.hub.register <- client

		// Send welcome message
		welcomeMsg := WebSocketMessage{
			Type:    "welcome",
			Payload: json.RawMessage(`{"message":"Connected to Satonic WebSocket Server"}`),
		}
		welcomeBytes, _ := json.Marshal(welcomeMsg)
		client.send <- welcomeBytes

		// Allow collection of memory referenced by the caller by doing all work in
		// new goroutines
		go client.writePump()
		go client.readPump()
	}
}

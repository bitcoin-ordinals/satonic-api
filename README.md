# Satonic API

A Go backend API for an NFT auction platform with crypto wallet authentication and Bitcoin Ordinals support.

## Features

- User authentication with crypto wallets (Bitcoin)
- Alternative email-based authentication with verification codes
- Multi-wallet and multi-email support per user account
- Bitcoin Ordinals (NFT) management
- Auction system for NFTs with bidding support
- Real-time auction updates via WebSockets
- PSBT (Partially Signed Bitcoin Transaction) support for secure transfers

## Tech Stack

- Go (Golang)
- Chi Router
- PostgreSQL
- WebSockets for real-time communication
- JWT for authentication

## Getting Started

### Prerequisites

- Go 1.18+
- PostgreSQL 13+

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/satonic-api.git
   cd satonic-api
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Set up the PostgreSQL database:
   ```bash
   psql -U postgres -c "CREATE DATABASE satonic;"
   psql -U postgres -d satonic -f internal/store/schema.sql
   ```

4. Configure the application:
   Copy the config file and edit as needed:
   ```bash
   cp configs/config.json.example configs/config.json
   ```

5. Run the application:
   ```bash
   go run cmd/api/main.go
   ```

## API Endpoints

### Authentication

- `POST /api/auth/wallet-login` - Login with wallet signature
- `POST /api/auth/email-login` - Request email verification code
- `POST /api/auth/verify-code` - Verify email code and login
- `POST /api/auth/link-wallet` - Link a wallet to the user's account
- `POST /api/auth/link-email` - Link an email to the user's account

### NFTs

- `GET /api/nfts` - Get the authenticated user's NFTs
- `GET /api/nfts/{id}` - Get a specific NFT by ID

### Auctions

- `GET /api/auctions` - Get all auctions (with optional filters)
- `GET /api/auctions/{id}` - Get a specific auction by ID
- `POST /api/auctions` - Create a new auction
- `POST /api/auctions/{id}/finalize` - Finalize an auction

### WebSocket

- `GET /api/ws` - WebSocket connection for real-time auction updates and bidding

## WebSocket Messages

### Client to Server

- `{"type":"subscribe","payload":"AUCTION_ID"}` - Subscribe to an auction's updates
- `{"type":"unsubscribe","payload":"AUCTION_ID"}` - Unsubscribe from an auction's updates
- `{"type":"bid","payload":{"auction_id":"AUCTION_ID","wallet_id":"WALLET_ID","amount":1000000}}` - Place a bid

### Server to Client

- `{"type":"welcome","payload":{"message":"Connected to Satonic WebSocket Server"}}` - Welcome message
- `{"type":"auction_update","payload":{...}}` - Auction update notification
- `{"type":"bid_placed","payload":{...}}` - Confirmation of a successful bid
- `{"type":"error","payload":{"message":"Error message"}}` - Error notification

## Development

### Project Structure

```
├── cmd/
│   └── api/
│       └── main.go           # Application entry point
├── configs/
│   └── config.json           # Configuration file
├── internal/
│   ├── config/               # Configuration loading
│   ├── handlers/             # HTTP handlers
│   ├── models/               # Data models
│   ├── services/             # Business logic
│   └── store/                # Database interactions
└── pkg/                      # Reusable packages
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.


-alikaansahin
-mehmetbersanozgur

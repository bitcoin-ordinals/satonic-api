-- Create extension for UUID
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on user_id for faster lookups
CREATE INDEX IF NOT EXISTS wallets_user_id_idx ON wallets(user_id);
CREATE INDEX IF NOT EXISTS wallets_address_idx ON wallets(address);

-- Emails table
CREATE TABLE IF NOT EXISTS emails (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address TEXT NOT NULL UNIQUE,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on user_id for faster lookups
CREATE INDEX IF NOT EXISTS emails_user_id_idx ON emails(user_id);
CREATE INDEX IF NOT EXISTS emails_address_idx ON emails(address);

-- Email verification table
CREATE TABLE IF NOT EXISTS email_verifications (
    id UUID PRIMARY KEY,
    email_id UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on email_id for faster lookups
CREATE INDEX IF NOT EXISTS email_verifications_email_id_idx ON email_verifications(email_id);

-- NFTs table
CREATE TABLE IF NOT EXISTS nfts (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    token_id TEXT NOT NULL,
    inscription_id TEXT NOT NULL,
    collection TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    content_url TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    auction_id UUID
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS nfts_wallet_id_idx ON nfts(wallet_id);
CREATE INDEX IF NOT EXISTS nfts_token_id_idx ON nfts(token_id);
CREATE INDEX IF NOT EXISTS nfts_inscription_id_idx ON nfts(inscription_id);
CREATE INDEX IF NOT EXISTS nfts_collection_idx ON nfts(collection);
CREATE INDEX IF NOT EXISTS nfts_auction_id_idx ON nfts(auction_id);

-- Auctions table
CREATE TABLE IF NOT EXISTS auctions (
    id UUID PRIMARY KEY,
    nft_id UUID NOT NULL REFERENCES nfts(id) ON DELETE CASCADE,
    seller_wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    start_price BIGINT NOT NULL,
    reserve_price BIGINT,
    buy_now_price BIGINT,
    current_bid BIGINT,
    current_bidder_id UUID REFERENCES users(id) ON DELETE SET NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    psbt TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS auctions_nft_id_idx ON auctions(nft_id);
CREATE INDEX IF NOT EXISTS auctions_seller_wallet_id_idx ON auctions(seller_wallet_id);
CREATE INDEX IF NOT EXISTS auctions_status_idx ON auctions(status);
CREATE INDEX IF NOT EXISTS auctions_end_time_idx ON auctions(end_time);

-- Add foreign key from nfts to auctions
ALTER TABLE nfts
ADD CONSTRAINT nfts_auction_id_fkey
FOREIGN KEY (auction_id) REFERENCES auctions(id) ON DELETE SET NULL;

-- Bids table
CREATE TABLE IF NOT EXISTS bids (
    id UUID PRIMARY KEY,
    auction_id UUID NOT NULL REFERENCES auctions(id) ON DELETE CASCADE,
    bidder_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted BOOLEAN NOT NULL DEFAULT FALSE,
    signature TEXT
);

-- Create indexes for faster lookups
CREATE INDEX IF NOT EXISTS bids_auction_id_idx ON bids(auction_id);
CREATE INDEX IF NOT EXISTS bids_bidder_id_idx ON bids(bidder_id);
CREATE INDEX IF NOT EXISTS bids_wallet_id_idx ON bids(wallet_id);
CREATE INDEX IF NOT EXISTS bids_amount_idx ON bids(amount);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers to update updated_at column
CREATE TRIGGER update_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_wallets_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_emails_updated_at
BEFORE UPDATE ON emails
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_nfts_updated_at
BEFORE UPDATE ON nfts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auctions_updated_at
BEFORE UPDATE ON auctions
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column(); 
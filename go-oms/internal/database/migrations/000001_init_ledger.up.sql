CREATE TABLE IF NOT EXISTS balances (
    id SERIAL PRIMARY KEY,
    asset VARCHAR(20) NOT NULL UNIQUE,
    amount DECIMAL(18, 8) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS trades (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    price DECIMAL(18, 8) NOT NULL,
    quantity DECIMAL(18, 8) NOT NULL,
    status VARCHAR(20) NOT NULL,
    event_timestamp_ms BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_trades_timestamp ON trades(event_timestamp_ms);

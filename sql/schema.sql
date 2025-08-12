CREATE TABLE games
(
    name         TEXT NOT NULL,
    platform     TEXT NOT NULL,
    hash         TEXT,
    UNIQUE(name, platform, hash)
);

CREATE INDEX idx_hash ON games (hash);
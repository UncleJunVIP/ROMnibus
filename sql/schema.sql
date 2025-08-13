CREATE TABLE games
(
    name     TEXT NOT NULL,
    filename TEXT NOT NULL,
    platform TEXT NOT NULL,
    hash     TEXT,
    UNIQUE (filename, hash)
);

CREATE INDEX idx_hash ON games (hash);
CREATE INDEX idx_filename ON games (filename);
package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"romnibus/models"
)

var db *sql.DB

func InitDB(databasePath string) (*sql.DB, error) {
	var err error
	db, err = sql.Open("sqlite3", databasePath)

	return db, err
}

func CloseDB() error {
	return db.Close()
}

func FindByHash(db *sql.DB, hash string) (*models.Game, error) {
	if db == nil {
		return nil, errors.New("database is not initialized")
	}

	query := `SELECT name, filename, platform, hash FROM games WHERE LOWER(hash) = LOWER(?) LIMIT 1`

	var game models.Game
	err := db.QueryRow(query, hash).Scan(&game.Name, &game.Filename, &game.Platform, &game.Hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query game by hash: %w", err)
	}

	return &game, nil
}

func FindByFilename(db *sql.DB, filename string) (*models.Game, error) {
	if db == nil {
		return nil, errors.New("database is not initialized")
	}

	query := `SELECT name, filename, platform, hash FROM games WHERE LOWER(filename) = LOWER(?) LIMIT 1`

	var game models.Game
	err := db.QueryRow(query, filename).Scan(&game.Name, &game.Filename, &game.Platform, &game.Hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query game by filename: %w", err)
	}

	return &game, nil
}

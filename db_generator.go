package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	models "github.com/UncleJunVIP/ROMnibus/models"
	"github.com/UncleJunVIP/ROMnibus/utils"
	_ "github.com/mattn/go-sqlite3"
)

const (
	repoURL          = "https://github.com/libretro/libretro-database.git"
	tempDir          = "temp_libretro_db"
	databaseFilename = "ROMnibus.sqlite"
)

var datDirs = []string{
	"metadat/no-intro",
	"metadat/fbneo-split",
}

func main() {
	_ = os.Remove(databaseFilename)
	_, _ = os.Create(databaseFilename)

	db, err := utils.InitDB(databaseFilename)
	if err != nil {
		panic(err)
	}
	defer func(db *sql.DB) {
		_ = db.Close()
	}(db)

	_ = initDBSchema(db)
	populateDB(db)
}

func initDBSchema(db *sql.DB) error {
	schemaContent, err := os.ReadFile("sql/schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	_, err = db.Exec(string(schemaContent))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	fmt.Println("Database schema initialized successfully")
	return nil
}

func populateDB(db *sql.DB) {
	datFiles, err := downloadDATFiles()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	gameMap := make(map[string][]models.Game)

	for _, filePath := range datFiles {
		filename := filepath.Base(filePath)
		fmt.Printf("Processing %s...\n", filename)

		platform := parseFilename(filename)
		games, err := parseDAT(filePath, platform)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", filename, err)
			continue
		}

		fmt.Printf("Parsed %d games from %s\n", len(games), filename)
		gameMap[platform] = append(gameMap[platform], games...)
	}

	totalGames := 0
	for platform, games := range gameMap {
		fmt.Printf("Inserting %d games for platform: %s\n", len(games), platform)
		if err := insertGames(db, games); err != nil {
			panic(err)
		}
		totalGames += len(games)
	}

	fmt.Printf("Successfully inserted %d total games into database\n", totalGames)
}

func downloadDATFiles() ([]string, error) {
	_ = os.RemoveAll(tempDir)

	fmt.Println("Cloning libretro-database repository...")

	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	var datFiles []string

	for _, datDir := range datDirs {
		fmt.Printf("Processing directory %s...\n", datDir)

		datPath := filepath.Join(tempDir, datDir)

		if _, err := os.Stat(datPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("directory %s does not exist in repository", datDir)
		}

		err := filepath.Walk(datPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".dat") {
				datFiles = append(datFiles, path)
			}

			return nil
		})

		if err != nil {
			fmt.Printf("Error walking directory %s: %v\n", datDir, err)
		}
	}

	fmt.Printf("Found %d DAT files\n", len(datFiles))
	return datFiles, nil
}

func parseFilename(filename string) string {
	openIndex := strings.Index(filename, "(")
	if openIndex == -1 {
		name := strings.TrimSuffix(filename, ".dat")
		return strings.TrimSpace(name)
	}

	platform := strings.TrimSpace(filename[:openIndex])

	closeIndex := strings.Index(filename[openIndex:], ")")
	if closeIndex == -1 {
		return platform
	}

	return platform
}

func parseDAT(filename string, platform string) ([]models.Game, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var parsedGames []models.Game
	fileContent := string(content)

	gameBlockRegex := regexp.MustCompile(`(?s)game\s*\(\s*name\s+"([^"]+)"[^}]*?rom\s*\([^}]*?name\s+([^\s]+)[^}]*?sha1\s+([a-fA-F0-9]{40})[^}]*?\)\s*\)`)

	matches := gameBlockRegex.FindAllStringSubmatch(fileContent, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			gameName := match[1]
			filename := match[2]
			sha1Hash := match[3]

			if strings.HasPrefix(filename, "\"") {
				filename = ""
			}

			gameEntry := models.Game{
				Name:     gameName,
				Filename: filename,
				Platform: platform,
				Hash:     strings.ToLower(sha1Hash),
			}

			parsedGames = append(parsedGames, gameEntry)
		}
	}

	return parsedGames, nil
}

func insertGames(db *sql.DB, games []models.Game) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func(tx *sql.Tx) {
		_ = tx.Rollback()
	}(tx)

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO games (name, filename, platform, hash) 
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func(stmt *sql.Stmt) {
		_ = stmt.Close()
	}(stmt)

	for _, game := range games {
		_, err := stmt.Exec(game.Name, game.Filename, game.Platform, game.Hash)
		if err != nil {
			return fmt.Errorf("failed to insert game %s: %w", game.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

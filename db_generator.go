package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"hashymchashface/models"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	repoURL          = "https://github.com/libretro/libretro-database.git"
	datDir           = "metadat/no-intro"
	tempDir          = "temp_libretro_db"
	databaseFilename = "hash_database.sqlite"
)

func main() {
	os.Remove(databaseFilename)
	os.Create(databaseFilename)

	db, err := sql.Open("sqlite3", databaseFilename)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	initDBSchema(db)
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
	defer os.RemoveAll(tempDir)

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
	os.RemoveAll(tempDir)

	fmt.Println("Cloning libretro-database repository...")

	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	datPath := filepath.Join(tempDir, datDir)

	if _, err := os.Stat(datPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist in repository", datDir)
	}

	var datFiles []string
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
		return nil, fmt.Errorf("failed to find DAT files: %w", err)
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

	gameBlockRegex := regexp.MustCompile(`(?s)game\s*\(\s*name\s+"([^"]+)"[^}]*?rom\s*\([^}]*?sha1\s+([a-fA-F0-9]{40})[^}]*?\)\s*\)`)

	matches := gameBlockRegex.FindAllStringSubmatch(fileContent, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			gameName := match[1]
			sha1Hash := match[2]

			if gameName != "" && sha1Hash != "" {
				gameEntry := models.Game{
					Name:     gameName,
					Platform: platform,
					Hash:     strings.ToLower(sha1Hash),
				}
				parsedGames = append(parsedGames, gameEntry)
			}
		}
	}

	return parsedGames, nil
}

func insertGames(db *sql.DB, games []models.Game) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO games (name, platform, hash) 
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, game := range games {
		_, err := stmt.Exec(game.Name, game.Platform, game.Hash)
		if err != nil {
			return fmt.Errorf("failed to insert game %s: %w", game.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GameData represents the parsed game data
type GameData struct {
	Name     string
	Filename string
	Platform string
	ROMs     []ROM
}

// ROM represents a ROM with its hashes
type ROM struct {
	Name   string `json:"Name"`
	Size   int64  `json:"Size"`
	Crc    string `json:"Crc"`
	Md5    string `json:"Md5"`
	Sha1   string `json:"Sha1"`
	Sha256 string `json:"Sha256"`
}

// GameFile represents the structure of the JSON file
type GameFile struct {
	Id                   int    `json:"Id"`
	ObjectType           string `json:"ObjectType"`
	Name                 string `json:"Name"`
	SignatureDataObjects []struct {
		Name     string `json:"Name"`
		Year     string `json:"Year"`
		Platform string `json:"Platform"`
	} `json:"SignatureDataObjects"`
	Attributes []struct {
		AttributeName string          `json:"attributeName"`
		Value         json.RawMessage `json:"Value"`
	} `json:"Attributes"`
}

// ProcessFile processes a single JSON file
func ProcessFile(filename string) ([]GameData, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	var data GameFile
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("error parsing JSON in %s: %v", filename, err)
	}

	// Get platform from first signature
	var platform string
	if len(data.SignatureDataObjects) > 0 {
		platform = data.SignatureDataObjects[0].Platform
	}

	// Find ROMs in attributes
	var roms []ROM

	for _, attr := range data.Attributes {
		if attr.AttributeName == "ROMs" {
			// Handle different structures in Value field
			var romArray []interface{}

			// Try to unmarshal as array of objects
			if err := json.Unmarshal(attr.Value, &romArray); err == nil {
				// Process array of ROM objects
				for _, item := range romArray {
					if romObj, ok := item.(map[string]interface{}); ok {
						rom := ROM{}

						// Extract fields with proper type conversion
						if name, ok := romObj["Name"].(string); ok {
							rom.Name = name
						}
						if size, ok := romObj["Size"].(float64); ok {
							rom.Size = int64(size)
						}
						if crc, ok := romObj["Crc"].(string); ok {
							rom.Crc = crc
						}
						if md5, ok := romObj["Md5"].(string); ok {
							rom.Md5 = md5
						}
						if sha1, ok := romObj["Sha1"].(string); ok {
							rom.Sha1 = sha1
						}
						if sha256, ok := romObj["Sha256"].(string); ok {
							rom.Sha256 = sha256
						}

						// Only include ROMs with at least one hash populated
						if rom.Crc != "" || rom.Md5 != "" || rom.Sha1 != "" || rom.Sha256 != "" {
							roms = append(roms, rom)
						}
					}
				}
			} else {
				// Try to unmarshal as single object (for some files)
				var romObj map[string]interface{}
				if err := json.Unmarshal(attr.Value, &romObj); err == nil {
					rom := ROM{}

					// Extract fields with proper type conversion
					if name, ok := romObj["Name"].(string); ok {
						rom.Name = name
					}
					if size, ok := romObj["Size"].(float64); ok {
						rom.Size = int64(size)
					}
					if crc, ok := romObj["Crc"].(string); ok {
						rom.Crc = crc
					}
					if md5, ok := romObj["Md5"].(string); ok {
						rom.Md5 = md5
					}
					if sha1, ok := romObj["Sha1"].(string); ok {
						rom.Sha1 = sha1
					}
					if sha256, ok := romObj["Sha256"].(string); ok {
						rom.Sha256 = sha256
					}

					// Only include ROMs with at least one hash populated
					if rom.Crc != "" || rom.Md5 != "" || rom.Sha1 != "" || rom.Sha256 != "" {
						roms = append(roms, rom)
					}
				}
			}
		}
	}

	// Deduplicate ROMs
	roms = Deduplicate(roms)

	// Create game data
	game := GameData{
		Name:     data.Name,
		Filename: strings.TrimSuffix(filename, filepath.Ext(filename)),
		Platform: platform,
		ROMs:     roms,
	}

	// Handle multiple versions - if there are multiple signatures with different platforms
	// we create separate entries for each unique combination
	platforms := make(map[string]bool)

	// Add the main platform from first signature
	if platform != "" {
		platforms[platform] = true
	}

	// Add all unique platforms from signatures
	for _, sig := range data.SignatureDataObjects {
		if sig.Platform != "" {
			platforms[sig.Platform] = true
		}
	}

	// If we have multiple platforms, create separate entries
	if len(platforms) > 1 {
		var multiPlatformGames []GameData
		for plat := range platforms {
			game.Platform = plat
			multiPlatformGames = append(multiPlatformGames, game)
		}
		return multiPlatformGames, nil
	}

	return []GameData{game}, nil
}

// Deduplicate removes duplicate ROMs based on all hash fields
func Deduplicate(roms []ROM) []ROM {
	seen := make(map[string]bool)
	var unique []ROM

	for _, rom := range roms {
		// Create a key based on all hash fields
		key := fmt.Sprintf("%s:%s:%s:%s", rom.Crc, rom.Md5, rom.Sha1, rom.Sha256)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, rom)
		}
	}

	return unique
}

// MergeAndDeduplicate merges all game data and removes duplicates
func MergeAndDeduplicate(games []GameData) []GameData {
	seen := make(map[string]bool)
	var unique []GameData

	for _, game := range games {
		// Create a key based on name, filename, and platform
		key := fmt.Sprintf("%s:%s:%s", game.Name, game.Filename, game.Platform)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, game)
		}
	}

	return unique
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <path_to_json_files>")
	}

	path := os.Args[1]

	// Find all JSON files
	files, err := filepath.Glob(filepath.Join(path, "*.json"))
	if err != nil {
		log.Fatal("Error finding JSON files:", err)
	}

	var allGames []GameData

	for _, file := range files {
		games, err := ProcessFile(file)
		if err != nil {
			log.Printf("Error processing file %s: %v", file, err)
			continue
		}

		allGames = append(allGames, games...)
	}

	// Remove duplicates
	allGames = MergeAndDeduplicate(allGames)

	// Print results
	fmt.Printf("Found %d unique games with ROMs:\n\n", len(allGames))

	for i, game := range allGames {
		fmt.Printf("Game %d:\n", i+1)
		fmt.Printf("  Name: %s\n", game.Name)
		fmt.Printf("  Filename: %s\n", game.Filename)
		fmt.Printf("  Platform: %s\n", game.Platform)
		fmt.Printf("  ROMs: %d\n", len(game.ROMs))

		for j, rom := range game.ROMs {
			fmt.Printf("    ROM %d:\n", j+1)
			fmt.Printf("      Name: %s\n", rom.Name)
			fmt.Printf("      Size: %d\n", rom.Size)
			if rom.Crc != "" {
				fmt.Printf("      CRC: %s\n", rom.Crc)
			}
			if rom.Md5 != "" {
				fmt.Printf("      MD5: %s\n", rom.Md5)
			}
			if rom.Sha1 != "" {
				fmt.Printf("      SHA1: %s\n", rom.Sha1)
			}
			if rom.Sha256 != "" {
				fmt.Printf("      SHA256: %s\n", rom.Sha256)
			}
		}
		fmt.Println()
	}
}

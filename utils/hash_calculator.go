package utils

import (
	"archive/zip"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CalculateFileHash computes the SHA-1 hash of a specified file's contents or the first file in a zip archive.
// Returns the hash as a hexadecimal string or an error if hash computation fails.
func CalculateFileHash(filename string) (string, error) {
	if strings.ToLower(filepath.Ext(filename)) == ".zip" {
		return calculateZipHash(filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash for %s: %w", filename, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func calculateZipHash(filename string) (string, error) {
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file %s: %w", filename, err)
	}
	defer zipReader.Close()

	if len(zipReader.File) == 0 {
		return "", fmt.Errorf("zip file %s is empty", filename)
	}

	zipFile := zipReader.File[0]

	fileReader, err := zipFile.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file %s within zip: %w", zipFile.Name, err)
	}
	defer fileReader.Close()

	hasher := sha1.New()
	if _, err := io.Copy(hasher, fileReader); err != nil {
		return "", fmt.Errorf("failed to calculate hash for %s within zip: %w", zipFile.Name, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

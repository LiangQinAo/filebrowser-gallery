package fbhttp

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

var (
	rawCacheDB *sql.DB
	rawCacheDir = "./cache/raw_previews"
)

// InitRawCache initializes the SQLite database for RAW conversion records.
func InitRawCache(dbPath string) error {
	if dbPath == "" {
		dbPath = "./database/raw_cache.db"
	}
	
	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(rawCacheDir, 0755); err != nil {
		return err
	}

	var err error
	rawCacheDB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open raw cache db: %w", err)
	}

	_, err = rawCacheDB.Exec(`
		CREATE TABLE IF NOT EXISTS raw_cache (
			md5        TEXT NOT NULL,
			size       TEXT NOT NULL,
			cache_file TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			PRIMARY KEY (md5, size)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to init raw cache table: %w", err)
	}

	log.Printf("Raw cache initialized at %s", dbPath)
	return nil
}

// GetRawPreview returns the content of a RAW preview, using sips and SQLite cache.
func GetRawPreview(originalPath string, size PreviewSize) ([]byte, error) {
	if rawCacheDB == nil {
		return nil, fmt.Errorf("raw cache not initialized")
	}

	log.Printf("[RAW] Request: path=%s size=%v", originalPath, size.String())

	// Calculate MD5 of original file for caching
	sum, err := fileMD5(originalPath)
	if err != nil {
		log.Printf("[RAW] MD5 calculation failed: %v", err)
		return nil, err
	}

	sizeStr := size.String()
	
	// Check cache database
	var cachePath string
	err = rawCacheDB.QueryRow("SELECT cache_file FROM raw_cache WHERE md5 = ? AND size = ?", sum, sizeStr).Scan(&cachePath)
	if err == nil {
		// Found in DB, check if file exists
		if data, err := os.ReadFile(cachePath); err == nil {
			log.Printf("[RAW] Cache HIT: md5=%s", sum[:8])
			return data, nil
		}
		log.Printf("[RAW] Cache file missing on disk: %s", cachePath)
		// File missing, remove from DB
		rawCacheDB.Exec("DELETE FROM raw_cache WHERE md5 = ? AND size = ?", sum, sizeStr)
	}

	// Cache miss or file missing, convert with sips
	maxDim := 600
	if size == PreviewSizeBig {
		maxDim = 1920
	}

	outputPath := filepath.Join(rawCacheDir, fmt.Sprintf("%s_%s.jpg", sum, sizeStr))
	log.Printf("[RAW] Cache MISS. Converting: %s -> %s (max %dpx)", originalPath, outputPath, maxDim)

	args := []string{
		"-Z", strconv.Itoa(maxDim),
		"-s", "format", "jpeg",
		"--out", outputPath,
		originalPath,
	}
	
	cmd := exec.Command("sips", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[RAW] SIPS ERROR: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("sips failed: %w\noutput: %s", err, string(out))
	}

	// Read result
	data, err := os.ReadFile(outputPath)
	if err != nil {
		log.Printf("[RAW] Failed to read converted file: %v", err)
		return nil, err
	}

	// Save to DB
	_, _ = rawCacheDB.Exec(
		"INSERT OR REPLACE INTO raw_cache (md5, size, cache_file, created_at) VALUES (?, ?, ?, ?)",
		sum, sizeStr, outputPath, time.Now().Unix(),
	)

	log.Printf("[RAW] Successfully converted and saved to cache.")
	return data, nil
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// loadAPI returns the parsed RuntimeAPI, using a cached file when available and fresh enough.
func loadAPI(cacheDir string, noCache bool, maxAge time.Duration) (*RuntimeAPI, error) {
	cachePath := filepath.Join(cacheDir, cacheFileName)

	if !noCache {
		if api, err := loadFromCache(cachePath, maxAge); err == nil {
			fmt.Fprintf(os.Stderr, "using cached API data from %s\n", cachePath)
			return api, nil
		}
	}

	fmt.Fprintf(os.Stderr, "fetching API data from %s\n", runtimeAPIURL)
	data, err := fetchAPI()
	if err != nil {
		return nil, fmt.Errorf("fetching API: %w", err)
	}

	// Write cache
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write cache: %v\n", err)
	}

	var api RuntimeAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("parsing API JSON: %w", err)
	}
	return &api, nil
}

func loadFromCache(path string, maxAge time.Duration) (*RuntimeAPI, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if time.Since(info.ModTime()) > maxAge {
		return nil, fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var api RuntimeAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, err
	}
	return &api, nil
}

func fetchAPI() ([]byte, error) {
	resp, err := http.Get(runtimeAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

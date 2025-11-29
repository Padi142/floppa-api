package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Config struct {
	PocketBaseURL string `json:"pocketbase_url"`
}

func LoadConfig() (*Config, error) {
	// Try multiple config locations
	configPaths := []string{
		"/app/config.json",           // Mounted in Coolify/Docker
		"config/config.json",         // Local development
		"backend/config/config.json", // Explicit relative path
	}

	var file *os.File
	var err error
	var usedPath string

	for _, path := range configPaths {
		file, err = os.Open(path)
		if err == nil {
			usedPath = path
			break
		}
	}

	if file == nil {
		return nil, fmt.Errorf("failed to open config.json in any location: %w", err)
	}
	defer file.Close()

	log.Printf("Loading config from: %s", usedPath)

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config.json: %w", err)
	}

	return &config, nil
}

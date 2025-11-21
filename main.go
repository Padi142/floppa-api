package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	PocketBaseURL string `json:"pocketbase_url"`
}

func loadConfig() (*Config, error) {
	// Try multiple config locations
	configPaths := []string{
		"/config.json",  // Mounted in Coolify/Docker
		"config.json",   // Local development
		"./config.json", // Explicit relative path
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

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	r := gin.Default()
	
	// Serve frontend static files
	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/", "./frontend/dist/index.html")

	r.GET("/flopapi", func(c *gin.Context) {
		imagePath, err := getRandomImage()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.File(imagePath)
	})

	r.GET("/macka", func(c *gin.Context) {
		imageData, _, err := getRandomCat(context.Background(), config.PocketBaseURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Data(http.StatusOK, "image/jpeg", imageData)
	})

	r.GET("/macka/count", func(c *gin.Context) {
		count, err := getCollectionCount(context.Background(), config.PocketBaseURL, "macky")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"count": count})
	})

	log.Println("Server starting on :8080")
	r.Run(":8080")
}

func getRandomImage() (string, error) {
	floppaDir := "./floppa"

	files, err := os.ReadDir(floppaDir)
	if err != nil {
		return "", fmt.Errorf("failed to read floppa directory: %v", err)
	}

	var imageFiles []string
	for _, file := range files {
		if !file.IsDir() && isImageFile(file.Name()) {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	if len(imageFiles) == 0 {
		return "", fmt.Errorf("no image files found in floppa directory")
	}

	randomIndex := rand.Intn(len(imageFiles))
	selectedImage := imageFiles[randomIndex]

	return filepath.Join(floppaDir, selectedImage), nil
}

type CatRecord struct {
	ID    string `json:"id"`
	Image string `json:"image"` // The filename
}

type RandomCatsResponse struct {
	Items []CatRecord `json:"items"`
}

func getRandomCat(ctx context.Context, pocketBaseURL string) ([]byte, string, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/collections/macky/records?perPage=1&sort=@random", pocketBaseURL), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch random record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var randomResp RandomCatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&randomResp); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(randomResp.Items) == 0 {
		return nil, "", fmt.Errorf("no cat records found in collection")
	}

	cat := randomResp.Items[0]
	if cat.Image == "" {
		return nil, "", fmt.Errorf("record has no image field")
	}

	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/files/macky/%s/%s", pocketBaseURL, cat.ID, cat.Image), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create image request: %w", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("image download error %d: %s", resp.StatusCode, string(body))
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, cat.Image, nil
}

type CollectionStats struct {
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
}

func getCollectionCount(ctx context.Context, pocketBaseURL, collectionName string) (int, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/collections/%s/records?perPage=1", pocketBaseURL, collectionName), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var stats CollectionStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return stats.TotalItems, nil
}

func isImageFile(filename string) bool {
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return true
	default:
		return false
	}
}

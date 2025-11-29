package main

import (
	"bytes"
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

	. "backend/config"
)


func main() {
	cfg, err := config.loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	r := gin.Default()

	// Serve frontend static files
	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/", "./frontend/dist/index.html")

	r.GET("/floppapi", func(c *gin.Context) {
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
		imageData, cat, err := getRandomImageFromCollection(context.Background(), "macky", cfg.PocketBaseURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Update views in background to not block response
		go func(recordID string, currentViews int) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := updateRecordViews(ctx, cfg.PocketBaseURL, "macky", recordID, currentViews); err != nil {
				log.Printf("Failed to update views for record %s: %v", recordID, err)
			}
		}(cat.ID, cat.Views)

		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Data(http.StatusOK, "image/jpeg", imageData)
	})

	r.GET("/macka/vim/:vimId", func(c *gin.Context) {
		vimId := c.Param("vimId")
		imageData, err := getRandomImageFromCollectionByVIMId(context.Background(), "macky", vimId, cfg.PocketBaseURL)
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
		count, err := getCollectionCount(context.Background(), cfg.PocketBaseURL, "macky")
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
	Image string `json:"image"`
	Views int    `json:"views"`
}

type RandomCatsResponse struct {
	Items []CatRecord `json:"items"`
}

func getRandomImageFromCollection(ctx context.Context, collectionName, pocketBaseURL string) ([]byte, CatRecord, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/collections/%s/records?perPage=1&sort=@random", pocketBaseURL, collectionName), nil)
	if err != nil {
		return nil, CatRecord{}, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, CatRecord{}, fmt.Errorf("failed to fetch random record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, CatRecord{}, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var randomResp RandomCatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&randomResp); err != nil {
		return nil, CatRecord{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(randomResp.Items) == 0 {
		return nil, CatRecord{}, fmt.Errorf("no cat records found in collection")
	}

	cat := randomResp.Items[0]
	if cat.Image == "" {
		return nil, CatRecord{}, fmt.Errorf("record has no image field")
	}

	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/files/%s/%s/%s", pocketBaseURL, collectionName, cat.ID, cat.Image), nil)
	if err != nil {
		return nil, CatRecord{}, fmt.Errorf("failed to create image request: %w", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, CatRecord{}, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, CatRecord{}, fmt.Errorf("image download error %d: %s", resp.StatusCode, string(body))
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, CatRecord{}, fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, cat, nil
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

func updateRecordViews(ctx context.Context, pocketBaseURL, collectionName, recordID string, views int) error {
	payload := map[string]int{"views": views + 1}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", fmt.Sprintf("%s/api/collections/%s/records/%s", pocketBaseURL, collectionName, recordID), bytes.NewBuffer(bodyBytes))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func getRandomImageFromCollectionByVIMId(ctx context.Context, collectionName, vimId, pocketBaseURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/collections/%s/records?filter=vimId='%s'&sort=@random", pocketBaseURL, collectionName, vimId), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch random record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var randomResp RandomCatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&randomResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(randomResp.Items) == 0 {
		return nil, fmt.Errorf("no cat records found in collection")
	}

	cat := randomResp.Items[0]
	if cat.Image == "" {
		return nil, fmt.Errorf("record has no image field")
	}

	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/files/%s/%s/%s", pocketBaseURL, collectionName, cat.ID, cat.Image), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create image request: %w", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("image download error %d: %s", resp.StatusCode, string(body))
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, nil
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

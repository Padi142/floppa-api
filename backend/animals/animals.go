package animals

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
)

// Animal interface for different animal types
type Animal interface {
	GetRandomImage(ctx context.Context) ([]byte, error)
	GetCount(ctx context.Context) (int, error)
	GetName() string
}

// Record represents a PocketBase record
type Record struct {
	ID    string `json:"id"`
	Image string `json:"image"`
	Views int    `json:"views"`
}

type listResponse struct {
	Items      []Record `json:"items"`
	TotalItems int      `json:"totalItems"`
}

// LocalAnimal serves images from local filesystem
type LocalAnimal struct {
	Name      string
	Directory string
}

func (a *LocalAnimal) GetName() string {
	return a.Name
}

func (a *LocalAnimal) GetRandomImage(ctx context.Context) ([]byte, error) {
	files, err := os.ReadDir(a.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s directory: %w", a.Name, err)
	}

	var imageFiles []string
	for _, file := range files {
		if !file.IsDir() && isImageFile(file.Name()) {
			imageFiles = append(imageFiles, file.Name())
		}
	}

	if len(imageFiles) == 0 {
		return nil, fmt.Errorf("no image files found in %s directory", a.Name)
	}

	randomIndex := rand.Intn(len(imageFiles))
	selectedImage := imageFiles[randomIndex]

	return os.ReadFile(filepath.Join(a.Directory, selectedImage))
}

func (a *LocalAnimal) GetCount(ctx context.Context) (int, error) {
	files, err := os.ReadDir(a.Directory)
	if err != nil {
		return 0, fmt.Errorf("failed to read %s directory: %w", a.Name, err)
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() && isImageFile(file.Name()) {
			count++
		}
	}
	return count, nil
}

// PocketBaseAnimal serves images from PocketBase
type PocketBaseAnimal struct {
	Name           string
	CollectionName string
	PocketBaseURL  string
}

func (a *PocketBaseAnimal) GetName() string {
	return a.Name
}

func (a *PocketBaseAnimal) GetRandomImage(ctx context.Context) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/collections/%s/records?perPage=1&sort=@random", a.PocketBaseURL, a.CollectionName), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch random record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var listResp listResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(listResp.Items) == 0 {
		return nil, fmt.Errorf("no records found in collection")
	}

	record := listResp.Items[0]
	if record.Image == "" {
		return nil, fmt.Errorf("record has no image field")
	}

	// Update views in background
	go a.updateViews(record.ID, record.Views)

	// Download image
	req, err = http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/files/%s/%s/%s", a.PocketBaseURL, a.CollectionName, record.ID, record.Image), nil)
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

	return io.ReadAll(resp.Body)
}

func (a *PocketBaseAnimal) GetCount(ctx context.Context) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/collections/%s/records?perPage=1", a.PocketBaseURL, a.CollectionName), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var listResp listResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.TotalItems, nil
}

func (a *PocketBaseAnimal) updateViews(recordID string, currentViews int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload := map[string]int{"views": currentViews + 1}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal payload: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH",
		fmt.Sprintf("%s/api/collections/%s/records/%s", a.PocketBaseURL, a.CollectionName, recordID),
		bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to update views for record %s: %v", recordID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("API error updating views %d: %s", resp.StatusCode, string(body))
	}
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

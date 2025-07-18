package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
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

func isImageFile(filename string) bool {
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return true
	default:
		return false
	}
}

package handlers

import (
	"context"
	"net/http"

	"floppa-api/animals"

	"github.com/gin-gonic/gin"
)

// AnimalHandler handles requests for a specific animal
type AnimalHandler struct {
	Animal animals.Animal
}

// GetRandomImage returns a random image for the animal
func (h *AnimalHandler) GetRandomImage(c *gin.Context) {
	imageData, err := h.Animal.GetRandomImage(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	c.Data(http.StatusOK, "image/jpeg", imageData)
}

// GetCount returns the count of images for the animal
func (h *AnimalHandler) GetCount(c *gin.Context) {
	count, err := h.Animal.GetCount(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetMackaByVimId returns a handler for getting macka images by VIM ID (for iOS shortcut)
func GetMackaByVimId(pocketBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		vimId := c.Param("vimId")
		imageData, err := animals.GetImageByVimId(context.Background(), "macky", vimId, pocketBaseURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		c.Data(http.StatusOK, "image/jpeg", imageData)
	}
}

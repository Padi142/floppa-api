package main

import (
	"log"
	"net/http"

	"floppa-api/animals"
	"floppa-api/config"
	"floppa-api/handlers"

	"github.com/gin-gonic/gin"
)

// AnimalConfig defines an animal endpoint - add new animals here!
type AnimalConfig struct {
	Endpoint    string // URL path (e.g., "macka")
	Title       string // Display name (e.g., "Macka")
	Description string // Short description
	Animal      animals.Animal
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Define all animals here - easy to add new ones!
	animalConfigs := []AnimalConfig{
		{
			Endpoint:    "floppapi",
			Title:       "Floppa Generator 3000",
			Description: "Klikni a uvidis zazrak!",
			Animal: &animals.LocalAnimal{
				Name:      "floppa",
				Directory: "./floppa",
			},
		},
		{
			Endpoint:    "macka",
			Title:       "Macka (z epicke macka databaze)",
			Description: "Originalna macka z databazy!",
			Animal: &animals.PocketBaseAnimal{
				Name:           "macka",
				CollectionName: "macky",
				PocketBaseURL:  cfg.PocketBaseURL,
			},
		},
		{
			Endpoint:    "capybara",
			Title:       "Capybara",
			Description: "OK I pull up",
			Animal: &animals.PocketBaseAnimal{
				Name:           "capybara",
				CollectionName: "capybaras",
				PocketBaseURL:  cfg.PocketBaseURL,
			},
		},
	}

	r := gin.Default()

	// Serve frontend static files
	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/", "./frontend/dist/index.html")

	// Register all animal endpoints
	for _, ac := range animalConfigs {
		handler := &handlers.AnimalHandler{Animal: ac.Animal}
		r.GET("/"+ac.Endpoint, handler.GetRandomImage)
		r.GET("/"+ac.Endpoint+"/count", handler.GetCount)
	}

	// API endpoint to list available animals (for frontend)
	r.GET("/api/animals", func(c *gin.Context) {
		type animalInfo struct {
			Endpoint    string `json:"endpoint"`
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		result := make([]animalInfo, len(animalConfigs))
		for i, ac := range animalConfigs {
			result[i] = animalInfo{
				Endpoint:    ac.Endpoint,
				Title:       ac.Title,
				Description: ac.Description,
			}
		}
		c.JSON(http.StatusOK, result)
	})

	// Special endpoint for macka by VIM ID (for iOS shortcut)
	r.GET("/macka/vim/:vimId", handlers.GetMackaByVimId(cfg.PocketBaseURL))

	log.Println("Server starting on :8080")
	r.Run(":8080")
}

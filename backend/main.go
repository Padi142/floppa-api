package main

import (
	"log"

	"floppa-api/animals"
	"floppa-api/config"
	"floppa-api/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	r := gin.Default()

	// Serve frontend static files
	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/", "./frontend/dist/index.html")

	// Floppa - local images
	floppaHandler := &handlers.AnimalHandler{
		Animal: &animals.LocalAnimal{
			Name:      "floppa",
			Directory: "./floppa",
		},
	}
	r.GET("/floppapi", floppaHandler.GetRandomImage)
	r.GET("/floppapi/count", floppaHandler.GetCount)

	// Macka - PocketBase
	mackaHandler := &handlers.AnimalHandler{
		Animal: &animals.PocketBaseAnimal{
			Name:           "macka",
			CollectionName: "macky",
			PocketBaseURL:  cfg.PocketBaseURL,
		},
	}
	r.GET("/macka", mackaHandler.GetRandomImage)
	r.GET("/macka/count", mackaHandler.GetCount)

	// Capybara - PocketBase
	capybaraHandler := &handlers.AnimalHandler{
		Animal: &animals.PocketBaseAnimal{
			Name:           "capybara",
			CollectionName: "capybaras",
			PocketBaseURL:  cfg.PocketBaseURL,
		},
	}
	r.GET("/capybara", capybaraHandler.GetRandomImage)
	r.GET("/capybara/count", capybaraHandler.GetCount)

	log.Println("Server starting on :8080")
	r.Run(":8080")
}

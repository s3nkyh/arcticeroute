package main

import (
	"ArcticeRoute/internal/services/ice"
	"github.com/gin-gonic/gin"
	"internal/services/ice/ice.go"
	"log"
	"net/http"
)

func main() {
	r := gin.Default()

	loader := ice.NewIceLoader()

	log.Println("Downloading OSI-SAF ice data...")
	if err := loader.DownloadIceData(); err != nil {
		log.Println("Warning: failed to download ice data:", err)
	} else {
		log.Println("Ice data downloaded successfully.")
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/ice", func(c *gin.Context) {
		data, err := loader.LoadCached()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no cached ice data"})
			return
		}

		c.Data(200, "application/json", data)
	})

	r.Run(":8080")
}

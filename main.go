package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/s3nkyh/arcticeroute/api"
)

func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/ships", getShips)
		apiGroup.GET("/glaciers", getGlaciers)
		apiGroup.GET("/health", healthCheck)
	}

	r.Static("/css", "./frontend")
	r.Static("/js", "./frontend")
	r.Static("/assets", "./frontend")

	r.LoadHTMLFiles("./frontend/index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	port := ":8080"
	log.Printf("🚀 Server starting on http://localhost%s", port)
	log.Printf("📍 Frontend: http://localhost%s", port)

	if err := r.Run(port); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getShips(c *gin.Context) {
	ships := api.Get10Ships()
	c.JSON(200, ships)
}

func getGlaciers(c *gin.Context) {
	bbox := c.DefaultQuery("bbox", "65,30,90,180")
	glaciers, err := api.GetGlaciers(bbox)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, glaciers)
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"framework": "Gin",
	})
}

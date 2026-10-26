package main

import (
	"log"

	"coffee_paste_backend/config"
	"coffee_paste_backend/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan, menggunakan environment variable")
	}

	cfg := config.LoadConfig()

	db, err := config.ConnectDatabase(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()

	routes.SetupRoutes(router, db, &cfg)

	log.Println("================================")
	log.Println("   COFFEE PASTE BACKEND")
	log.Println("http://localhost:8080/api/v1/health")
	log.Println("================================")
	log.Println("Database:", cfg.DBName)
	log.Println("Server berjalan di port:", cfg.AppPort)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal("Gagal menjalankan server:", err)
	}
}

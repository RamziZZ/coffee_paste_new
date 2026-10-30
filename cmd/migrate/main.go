package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"coffee_paste_backend/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan, menggunakan environment variable")
	}

	cfg := config.LoadConfig()

	db, err := connectDB(cfg)
	if err != nil {
		log.Fatal("Gagal koneksi database:", err)
	}
	defer db.Close()

	migrationsDir := "database/migrations"
	if len(os.Args) > 1 {
		migrationsDir = os.Args[1]
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatal("Gagal membaca direktori migrations:", err)
	}

	ctx := context.Background()

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Gagal membaca %s: %v", path, err)
			continue
		}

		_, err = db.Exec(ctx, string(sqlBytes))
		if err != nil {
			log.Printf("Gagal menjalankan %s: %v", entry.Name(), err)
		} else {
			fmt.Printf("✓ %s OK\n", entry.Name())
		}
	}

	fmt.Println("\nMigrasi selesai.")
}

func connectDB(cfg config.Config) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	return pgxpool.New(context.Background(), connStr)
}

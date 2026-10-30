package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"coffee_paste_backend/config"
	"coffee_paste_backend/pkg/password"

	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedUser adalah spesifikasi satu user yang akan dibuat.
type SeedUser struct {
	Name     string
	Email    string
	Role     string
	Password string
}

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

	ctx := context.Background()

	users := []SeedUser{
		{
			Name:     "Ramzi Farhan",
			Email:    "ramzifarhan072@gmail.com",
			Role:     "customer",
			Password: "CoffeePaste123!",
		},
		{
			Name:     "Gaming Sniper",
			Email:    "gamingsniper094@gmail.com",
			Role:     "admin",
			Password: "CoffeePaste123!",
		},
	}

	for _, u := range users {
		if err := upsertUser(ctx, db, u); err != nil {
			log.Printf("Gagal membuat user %s: %v", u.Email, err)
			os.Exit(1)
		}
		fmt.Printf("✓ User %s (%s) siap\n", u.Email, u.Role)
	}

	fmt.Println("\nSeed selesai. Login dengan password: CoffeePaste123!")
}

func connectDB(cfg config.Config) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	return pgxpool.New(context.Background(), connStr)
}

func upsertUser(ctx context.Context, db *pgxpool.Pool, u SeedUser) error {
	hash, err := password.Hash(u.Password)
	if err != nil {
		return err
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Cek apakah email sudah ada
	var existingID string
	err = tx.QueryRow(
		ctx,
		`SELECT id::text FROM users WHERE LOWER(email) = LOWER($1)`,
		u.Email,
	).Scan(&existingID)

	var userID string

	if err == nil {
		// Update user existing
		userID = existingID
		_, err = tx.Exec(
			ctx,
			`UPDATE users SET
				role = $1,
				name = $2,
				password_hash = $3,
				is_active = TRUE
			 WHERE id = $4`,
			u.Role,
			u.Name,
			hash,
			userID,
		)
		if err != nil {
			return err
		}
	} else {
		// Insert user baru
		err = tx.QueryRow(
			ctx,
			`INSERT INTO users (id, role, name, email, password_hash, is_active)
			 VALUES (gen_random_uuid(), $1, $2, $3, $4, TRUE)
			 RETURNING id::text`,
			u.Role,
			u.Name,
			u.Email,
			hash,
		).Scan(&userID)
		if err != nil {
			return err
		}
	}

	// Untuk customer: pastikan baris customers ada
	if u.Role == "customer" {
		var custExists bool
		err = tx.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM customers WHERE user_id = $1)`,
			userID,
		).Scan(&custExists)
		if err != nil {
			return err
		}

		if !custExists {
			_, err = tx.Exec(
				ctx,
				`INSERT INTO customers (id, user_id, name, email)
				 VALUES (gen_random_uuid(), $1, $2, $3)`,
				userID,
				u.Name,
				u.Email,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

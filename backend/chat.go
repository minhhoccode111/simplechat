package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Message struct {
	ID      string
	Content string
	Created time.Time
	UserID  string
}

func NewPool(pgURL string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(pgURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 2
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func main() {
	fmt.Println("Hello, World! From chat app")

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	pgURL := os.Getenv("PG_URL")

	pool, err := NewPool(pgURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatal("DB unreachable:", err)
	}

	log.Printf("Database connected")
}

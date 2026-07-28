package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/schollz/cowyo2/internal/database"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}

	store, err := database.Open(context.Background(), database.ConfigFromEnv())
	if err != nil {
		return err
	}
	backend := store.Backend()
	if err := store.Close(); err != nil {
		return fmt.Errorf("close %s database: %w", backend, err)
	}

	log.Printf("%s database migrations are up to date", backend)
	return nil
}

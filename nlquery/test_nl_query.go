package nlquery

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func RunTests() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database configuration
	dbConfig := map[string]string{
		"host":     os.Getenv("DB_HOST"),
		"port":     os.Getenv("DB_PORT"),
		"user":     os.Getenv("DB_USER"),
		"password": os.Getenv("DB_PASSWORD"),
		"dbname":   os.Getenv("DB_NAME"),
	}

	// Initialize NL Query Engine
	engine, err := NewNLQueryEngine(dbConfig)
	if err != nil {
		log.Fatalf("Error initializing NL Query Engine: %v", err)
	}
	defer engine.Close()

	fmt.Println("Testing Natural Language Queries...")
	fmt.Println("===================================\n")

	// Test queries
	queries := []string{
		"What is the average score of candidates from each state?",
		"Show me the top 10 performing candidates",
		"How many candidates applied for medicine related courses?",
		"What is the gender distribution of candidates?",
	}

	// Process each query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, query := range queries {
		fmt.Printf("Query: %s\n", query)
		fmt.Println("-----------------------------------")
		if err := engine.ProcessQuery(ctx, query); err != nil {
			fmt.Printf("Error processing query: %v\n", err)
		}
		fmt.Println("-----------------------------------\n")
	}
}

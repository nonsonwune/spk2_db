package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Database configuration
	dbConfig := map[string]string{
		"host":     os.Getenv("DB_HOST"),
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

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test queries
	queries := []string{
		"What is the average score of candidates from each state?",
		"Show me the top 10 performing candidates",
		"How many candidates applied for medicine related courses?",
		"What is the gender distribution of candidates?",
	}

	fmt.Println("Testing Natural Language Queries...")
	fmt.Println("===================================")

	for _, query := range queries {
		fmt.Printf("\nQuery: %s\n", query)
		fmt.Println("-----------------------------------")
		
		if err := engine.ProcessQuery(ctx, query); err != nil {
			fmt.Printf("Error processing query: %v\n", err)
		}
		
		fmt.Println("-----------------------------------")
		time.Sleep(2 * time.Second) // Small delay between queries
	}
}

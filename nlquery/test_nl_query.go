package nlquery

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load("../.env"); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	os.Exit(m.Run())
}

func setupTestDB(t *testing.T) *sql.DB {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func setupGeminiClient(t *testing.T) *genai.GenerativeModel {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	temp := float32(0.2)
	model.Temperature = &temp

	return model
}

func TestNLQueryEngine_ProcessQuery(t *testing.T) {
	// Create a mock database connection
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error creating mock database: %v", err)
	}
	defer db.Close()

	// Set up mock expectations
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	// Create the NL query engine
	engine, err := NewNLQueryEngine(db)
	if err != nil {
		t.Fatalf("Error creating NL query engine: %v", err)
	}

	testCases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "Simple count query",
			query:   "How many students are there?",
			wantErr: false,
		},
		{
			name:    "Basic state query",
			query:   "How many students applied from Lagos state?",
			wantErr: false,
		},
		{
			name:    "Course specific query",
			query:   "Show me the top 5 courses by number of applicants",
			wantErr: false,
		},
		{
			name:    "Invalid query",
			query:   "What is the meaning of life?",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := engine.ProcessQuery(tc.query)
			if (err != nil) != tc.wantErr {
				t.Errorf("ProcessQuery() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && result == "" {
				t.Error("ProcessQuery() returned empty result")
			}
		})
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

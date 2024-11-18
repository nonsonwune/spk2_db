package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/api/option"
)

type NLQueryEngine struct {
	client *genai.Client
	model  *genai.GenerativeModel
	db     *sql.DB
}

// SQL query templates for common operations
var queryTemplates = map[string]string{
	"select_all":      "SELECT %s FROM %s",
	"count":           "SELECT COUNT(*) FROM %s",
	"average":         "SELECT AVG(%s) FROM %s",
	"group_by":        "SELECT %s, COUNT(*) FROM %s GROUP BY %s",
	"join":           "SELECT %s FROM %s JOIN %s ON %s",
	"filter":         "SELECT %s FROM %s WHERE %s",
	"order_by":       "SELECT %s FROM %s ORDER BY %s %s LIMIT %d",
	"correlation":    "SELECT corr(%s, %s) FROM %s",
}

// Initialize NLQueryEngine with database and Gemini API client
func NewNLQueryEngine(dbConfig map[string]string) (*NLQueryEngine, error) {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	// Initialize database connection
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbConfig["host"], dbConfig["user"], dbConfig["password"], dbConfig["dbname"])
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	// Initialize Gemini API client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return nil, fmt.Errorf("error initializing Gemini client: %v", err)
	}

	// Initialize model with SQL generation capabilities
	model := client.GenerativeModel("gemini-1.5-pro-latest")
	temp := float32(0.3)
	topk := int32(1)
	topp := float32(0.8)
	model.Temperature = &temp // Lower temperature for more precise SQL generation
	model.TopK = &topk       // More focused responses
	model.TopP = &topp

	// Set up function declarations for SQL operations
	model.Tools = []*genai.Tool{{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "generateSQL",
			Description: "Generate SQL query from natural language input",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"operation": {
						Type:        genai.TypeString,
						Description: "SQL operation type (select, count, average, etc.)",
					},
					"tables": {
						Type:        genai.TypeArray,
						Description: "Tables involved in the query",
						Items:       &genai.Schema{Type: genai.TypeString},
					},
					"columns": {
						Type:        genai.TypeArray,
						Description: "Columns to include in the query",
						Items:       &genai.Schema{Type: genai.TypeString},
					},
					"conditions": {
						Type:        genai.TypeString,
						Description: "WHERE clause conditions",
					},
				},
				Required: []string{"operation", "tables"},
			},
		}},
	}}

	return &NLQueryEngine{
		client: client,
		model:  model,
		db:     db,
	}, nil
}

// Process natural language query
func (e *NLQueryEngine) ProcessQuery(ctx context.Context, query string) error {
	// Start chat session
	chat := e.model.StartChat()

	// Send the query to Gemini
	prompt := fmt.Sprintf(`As a SQL expert, analyze this query and generate a SQL statement for our JAMB database:
Query: %s

Consider:
1. Tables: candidates, candidate_scores, subjects
2. Common columns: reg_number, state_code, score, subject_code
3. Use appropriate joins and conditions
4. Ensure statistical validity for aggregations
5. Format results clearly

Return the SQL query in a structured format that can be executed.`, query)

	resp, err := chat.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return fmt.Errorf("error sending message to Gemini: %v", err)
	}

	// Extract SQL query from response
	sqlQuery := e.extractSQLFromResponse(resp)
	if sqlQuery == "" {
		return fmt.Errorf("no valid SQL query generated")
	}

	// Execute the query
	results, err := e.executeQuery(sqlQuery)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}

	// Display results
	e.displayResults(results)
	return nil
}

// Extract SQL query from Gemini response
func (e *NLQueryEngine) extractSQLFromResponse(resp *genai.GenerateContentResponse) string {
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				// Look for SQL query in the response
				if idx := strings.Index(string(text), "SELECT"); idx != -1 {
					query := string(text)[idx:]
					if endIdx := strings.Index(query, ";"); endIdx != -1 {
						return query[:endIdx]
					}
					return query
				}
			}
		}
	}
	return ""
}

// Execute SQL query and return results
func (e *NLQueryEngine) executeQuery(query string) ([]map[string]interface{}, error) {
	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		result := make(map[string]interface{})
		columnPointers := make([]interface{}, len(columns))
		for i := range columns {
			columnPointers[i] = new(interface{})
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colName := range columns {
			val := columnPointers[i].(*interface{})
			result[colName] = *val
		}
		results = append(results, result)
	}

	return results, nil
}

// Display results in a formatted table
func (e *NLQueryEngine) displayResults(results []map[string]interface{}) {
	if len(results) == 0 {
		fmt.Println("No results found")
		return
	}

	// Get column names from first result
	var columns []string
	for col := range results[0] {
		columns = append(columns, col)
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(columns)

	// Add rows
	for _, result := range results {
		var row []string
		for _, col := range columns {
			val := fmt.Sprintf("%v", result[col])
			row = append(row, val)
		}
		table.Append(row)
	}

	table.Render()
}

// Close resources
func (e *NLQueryEngine) Close() {
	if e.db != nil {
		e.db.Close()
	}
	if e.client != nil {
		e.client.Close()
	}
}

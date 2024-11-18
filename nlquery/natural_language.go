package nlquery

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/api/option"

	"github.com/nonsonwune/spk2_db/nlquery/prompts"
)

type NLQueryEngine struct {
	client  *genai.Client
	model   *genai.GenerativeModel
	db      *sql.DB
	prompts *prompts.PromptBuilder
}

// Initialize NLQueryEngine with database and Gemini API client
func NewNLQueryEngine(dbConfig map[string]string) (*NLQueryEngine, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbConfig["host"], dbConfig["user"], dbConfig["password"], dbConfig["dbname"])
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		return nil, fmt.Errorf("error initializing Gemini client: %v", err)
	}

	model := client.GenerativeModel("gemini-1.5-pro-latest")
	temp := float32(0.2) // Lower temperature for more precise SQL
	model.Temperature = &temp

	return &NLQueryEngine{
		client:  client,
		model:   model,
		db:      db,
		prompts: prompts.NewPromptBuilder(),
	}, nil
}

// Process natural language query
func (e *NLQueryEngine) ProcessQuery(ctx context.Context, query string) error {
	queryCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	// Generate SQL query using Gemini
	sqlQuery, err := e.generateSQLQuery(queryCtx, query)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return fmt.Errorf("The query timed out. Try a more specific question or add more filters (e.g., year, state, course)")
		}
		errMsg, _ := e.getErrorMessage(queryCtx, query, err)
		return fmt.Errorf(errMsg)
	}

	// Validate the generated query
	if valid, reason := e.validateQuery(queryCtx, query, sqlQuery); !valid {
		return fmt.Errorf("invalid query: %s", reason)
	}

	fmt.Printf("\nExecuting SQL Query:\n%s\n\n", sqlQuery)
	results, err := e.executeQuery(sqlQuery)
	if err != nil {
		errMsg, _ := e.getErrorMessage(queryCtx, query, err)
		return fmt.Errorf(errMsg)
	}

	e.displayResults(results)
	return nil
}

func (e *NLQueryEngine) generateSQLQuery(ctx context.Context, query string) (string, error) {
	var sqlQuery string
	var err error

	for retries := 3; retries > 0; retries-- {
		chat := e.model.StartChat()
		prompt := e.prompts.BuildQueryPrompt(query)
		
		resp, err := chat.SendMessage(ctx, genai.Text(prompt))
		if err == nil && len(resp.Candidates) > 0 {
			text := resp.Candidates[0].Content.Parts[0]
			if textStr, ok := text.(genai.Text); ok {
				sqlQuery = string(textStr)
				sqlQuery = strings.TrimSpace(sqlQuery)
				if strings.HasPrefix(sqlQuery, "```sql") {
					sqlQuery = strings.TrimPrefix(sqlQuery, "```sql")
					sqlQuery = strings.TrimSuffix(sqlQuery, "```")
				}
				sqlQuery = strings.TrimSpace(sqlQuery)
				break
			}
		}
		time.Sleep(time.Second)
	}

	if err != nil || sqlQuery == "" {
		return "", fmt.Errorf("failed to generate SQL query: %v", err)
	}

	return sqlQuery, nil
}

func (e *NLQueryEngine) validateQuery(ctx context.Context, query, sql string) (bool, string) {
	chat := e.model.StartChat()
	prompt := e.prompts.BuildValidationPrompt(query, sql)
	
	resp, err := chat.SendMessage(ctx, genai.Text(prompt))
	if err != nil || len(resp.Candidates) == 0 {
		return false, "validation failed due to API error"
	}

	text := resp.Candidates[0].Content.Parts[0]
	if textStr, ok := text.(genai.Text); ok {
		result := strings.TrimSpace(string(textStr))
		if strings.HasPrefix(result, "VALID") {
			return true, ""
		}
		if strings.HasPrefix(result, "INVALID: ") {
			return false, strings.TrimPrefix(result, "INVALID: ")
		}
		return false, fmt.Sprintf("validation failed: %s", result)
	}
	return false, "invalid response format from validation"
}

func (e *NLQueryEngine) getErrorMessage(ctx context.Context, query string, err error) (string, error) {
	chat := e.model.StartChat()
	prompt := e.prompts.BuildErrorPrompt(query, err)
	
	resp, err := chat.SendMessage(ctx, genai.Text(prompt))
	if err != nil || len(resp.Candidates) == 0 {
		return "An error occurred while processing your query", nil
	}

	text := resp.Candidates[0].Content.Parts[0]
	if textStr, ok := text.(genai.Text); ok {
		return strings.TrimSpace(string(textStr)), nil
	}
	return "An error occurred while processing your query", nil
}

// Execute SQL query and return results
func (e *NLQueryEngine) executeQuery(query string) ([]map[string]interface{}, error) {
	// Increase timeout to 30 seconds for large queries
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Add query optimization hints for COUNT queries
	if strings.Contains(strings.ToUpper(query), "COUNT(") {
		// Use EXPLAIN to check if we need table scan
		explain := "EXPLAIN " + query
		row := e.db.QueryRowContext(ctx, explain)
		var plan string
		if err := row.Scan(&plan); err == nil {
			if strings.Contains(strings.ToLower(plan), "seq scan") {
				// Add PARALLEL hint for large table scans
				query = strings.Replace(query, "SELECT", "SELECT /*+ PARALLEL(4) */", 1)
			}
		}
	}

	rows, err := e.db.QueryContext(ctx, query)
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
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		for i, column := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				result[column] = string(b)
			} else {
				result[column] = val
			}
		}
		results = append(results, result)
	}

	return results, nil
}

// Display results in a table format
func (e *NLQueryEngine) displayResults(results []map[string]interface{}) {
	if len(results) == 0 {
		fmt.Println("No results found")
		return
	}

	// Get columns from the first result
	var columns []string
	for column := range results[0] {
		columns = append(columns, column)
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(columns)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	// Add rows
	for _, result := range results {
		var row []string
		for _, column := range columns {
			value := result[column]
			if value == nil {
				row = append(row, "NULL")
			} else {
				row = append(row, fmt.Sprintf("%v", value))
			}
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

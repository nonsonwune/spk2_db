package nlquery

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
	"regexp"
	"strconv"

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
    // Create context with longer timeout
    queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    // Try common query patterns first
    if sqlQuery := e.matchCommonPattern(query); sqlQuery != "" {
        fmt.Printf("\nUsing query template:\n%s\n\n", sqlQuery)
        results, err := e.executeQuery(sqlQuery)
        if err == nil {
            e.displayResults(results)
            return nil
        }
    }

    // Start chat session with retry mechanism
    var resp *genai.GenerateContentResponse
    var err error
    for retries := 3; retries > 0; retries-- {
        chat := e.model.StartChat()
        resp, err = chat.SendMessage(queryCtx, genai.Text(e.buildPrompt(query)))
        if err == nil {
            break
        }
        time.Sleep(time.Second) // Wait before retry
    }
    if err != nil {
        return fmt.Errorf("failed to generate query after retries: %v", err)
    }

    // Extract and execute SQL query
    sqlQuery := e.extractSQLFromResponse(resp)
    if sqlQuery == "" {
        sqlQuery = e.getFallbackQuery(query)
    }

    if sqlQuery == "" {
        return fmt.Errorf("could not generate SQL query - please try rephrasing your question")
    }

    fmt.Printf("\nExecuting SQL Query:\n%s\n\n", sqlQuery)
    results, err := e.executeQuery(sqlQuery)
    if err != nil {
        return fmt.Errorf("error executing query: %v", err)
    }

    e.displayResults(results)
    return nil
}

// Match common query patterns
func (e *NLQueryEngine) matchCommonPattern(query string) string {
    query = strings.ToLower(query)
    year := time.Now().Year()
    
    // Extract year if present
    if matches := regexp.MustCompile(`\b20\d{2}\b`).FindString(query); matches != "" {
        year, _ = strconv.Atoi(matches)
    }

    // Pattern 1: Count candidates for medicine
    if strings.Contains(query, "how many") && strings.Contains(query, "medicine") {
        return fmt.Sprintf(`
            SELECT COUNT(DISTINCT c.regnumber) as total_candidates
            FROM candidate c
            JOIN course co ON c.app_course1 = co.course_code
            WHERE LOWER(co.course_name) SIMILAR TO '%(medicine|medical|mbbs|pharmacy)%%'
            AND c.year = %d;`, year)
    }

    // Pattern 2: Count by gender and medicine
    if strings.Contains(query, "gender") && strings.Contains(query, "medicine") {
        return fmt.Sprintf(`
            SELECT COALESCE(c.gender, 'Unknown') as gender,
                   COUNT(DISTINCT c.regnumber) as total_candidates
            FROM candidate c
            JOIN course co ON c.app_course1 = co.course_code
            WHERE LOWER(co.course_name) SIMILAR TO '%(medicine|medical|mbbs|pharmacy)%%'
            AND c.year = %d
            GROUP BY c.gender
            ORDER BY total_candidates DESC;`, year)
    }

    // Pattern 3: Count institutions and admissions
    if strings.Contains(query, "institution") && strings.Contains(query, "admit") {
        return fmt.Sprintf(`
            SELECT COUNT(DISTINCT i.inid) as total_institutions,
                   COUNT(DISTINCT c.regnumber) as total_admissions
            FROM candidate c
            JOIN institution i ON c.inid = i.inid
            WHERE c.is_admitted = true
            AND c.year = %d;`, year)
    }

    // Pattern 4: Institution admission statistics
    if strings.Contains(query, "institution") {
        return fmt.Sprintf(`
            SELECT i.inname as institution_name,
                   COUNT(DISTINCT c.regnumber) as admitted_candidates
            FROM candidate c
            JOIN institution i ON c.inid = i.inid
            WHERE c.is_admitted = true
            AND c.year = %d
            GROUP BY i.inid, i.inname
            ORDER BY admitted_candidates DESC
            LIMIT 20;`, year)
    }

    return ""
}

// Get fallback query for common questions
func (e *NLQueryEngine) getFallbackQuery(query string) string {
    query = strings.ToLower(query)
    year := time.Now().Year()
    
    // Extract year if present
    if matches := regexp.MustCompile(`\b20\d{2}\b`).FindString(query); matches != "" {
        year, _ = strconv.Atoi(matches)
    }

    // Default to institution statistics if query mentions institutions
    if strings.Contains(query, "institution") {
        return fmt.Sprintf(`
            SELECT 
                COUNT(DISTINCT i.inid) as total_institutions,
                COUNT(DISTINCT c.regnumber) as total_candidates,
                COUNT(DISTINCT CASE WHEN c.is_admitted THEN c.regnumber END) as admitted_candidates
            FROM candidate c
            JOIN institution i ON c.inid = i.inid
            WHERE c.year = %d;`, year)
    }

    // Default to medicine statistics
    return fmt.Sprintf(`
        SELECT COUNT(DISTINCT c.regnumber) as total_candidates
        FROM candidate c
        JOIN course co ON c.app_course1 = co.course_code
        WHERE LOWER(co.course_name) SIMILAR TO '%(medicine|medical|mbbs|pharmacy)%%'
        AND c.year = %d;`, year)
}

// Build AI prompt
func (e *NLQueryEngine) buildPrompt(query string) string {
    return fmt.Sprintf(`Generate a PostgreSQL query for: "%s"

Use these guidelines:
1. Always use COUNT(DISTINCT) for counting unique records
2. Use proper table aliases:
   - c for candidate
   - co for course
   - i for institution
   - s for state
3. Handle NULL values with COALESCE
4. Include year in conditions
5. Use proper JOINs
6. Limit results to 20 rows unless specified otherwise

Tables:
candidate (c):
  regnumber (PK), gender, statecode, aggregate, app_course1, year, is_admitted, inid
course (co):
  course_code (PK), course_name, facid
institution (i):
  inid (PK), inname, inabv, inst_state_id
state (s):
  st_id (PK), st_name

Return ONLY the SQL query.`, query)
}

// Extract SQL query from Gemini response
func (e *NLQueryEngine) extractSQLFromResponse(resp *genai.GenerateContentResponse) string {
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				// Clean up the response
				response := strings.TrimSpace(string(text))
				
				// If the response starts with a code block marker, extract the content
				if strings.HasPrefix(response, "```") {
					parts := strings.Split(response, "```")
					if len(parts) >= 3 {
						response = strings.TrimSpace(parts[1])
						// Remove the language identifier if present
						if idx := strings.Index(response, "\n"); idx != -1 {
							response = strings.TrimSpace(response[idx:])
						}
					}
				}

				// Look for SQL keywords
				sqlKeywords := []string{"SELECT", "WITH"}
				for _, keyword := range sqlKeywords {
					if idx := strings.Index(strings.ToUpper(response), keyword); idx != -1 {
						query := response[idx:]
						// Remove any trailing semicolon and extra whitespace
						query = strings.TrimRight(query, "; \n\t")
						return query
					}
				}
			}
		}
	}
	return ""
}

// Execute SQL query and return results
func (e *NLQueryEngine) executeQuery(query string) ([]map[string]interface{}, error) {
	// Add timeout to query execution
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("SQL error: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting columns: %v", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		result := make(map[string]interface{})
		columnPointers := make([]interface{}, len(columns))
		for i := range columns {
			columnPointers[i] = new(interface{})
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		for i, colName := range columns {
			val := columnPointers[i].(*interface{})
			if *val == nil {
				result[colName] = "NULL"
			} else {
				result[colName] = *val
			}
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
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

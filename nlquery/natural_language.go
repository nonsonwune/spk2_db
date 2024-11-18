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
	prompt := fmt.Sprintf(`As a SQL expert, generate a PostgreSQL query for our JAMB database based on this question:
"%s"

Complete Database Schema:
1. candidate table:
   - regnumber (PK, varchar(20))
   - surname, firstname, middlename (varchar(100))
   - gender (varchar(10))
   - statecode (int, FK -> state.st_id)
   - aggregate (int)
   - app_course1 (varchar(100), FK -> course.course_code)
   - inid (varchar(20), FK -> institution.inid)
   - lg_id (int, FK -> lga.lg_id)
   - year (int)
   - is_admitted (boolean)
   - is_direct_entry (boolean)
   - date_of_birth (date)

2. candidate_scores table:
   - cand_reg_number (PK/FK -> candidate.regnumber)
   - subject_id (PK/FK -> subject.su_id)
   - score (int)
   - year (PK, int)

3. course table:
   - course_code (PK, varchar(100))
   - course_name (varchar(200))
   - facid (int, FK -> faculty.fac_id)
   - course_abbreviation (varchar(50))
   - duration (int)
   - degree (varchar(50))

4. faculty table:
   - fac_id (PK, int)
   - fac_name (varchar(100))
   - fac_code (varchar(10))
   - fac_abv (varchar(20))

5. institution table:
   - inid (PK, varchar(20))
   - inname (varchar(200))
   - inabv (varchar(50))
   - inst_state_id (int, FK -> state.st_id)
   - inst_cat (varchar(20))

6. state table:
   - st_id (PK, int)
   - st_name (varchar(100))
   - st_abreviation (varchar(50))

7. lga table:
   - lg_id (PK, int)
   - lg_st_id (int, FK -> state.st_id)
   - lg_name (varchar(100))
   - lg_abreviation (varchar(50))

8. subject table:
   - su_id (PK, int)
   - su_name (varchar(100))
   - su_abrv (varchar(10))

Common Query Patterns:
1. Medical Courses:
   WHERE LOWER(course_name) SIMILAR TO '%(medic|health|nurs|pharm|dental|clinic|surg|therapy)%'

2. Location-based Queries:
   Join through state table:
   JOIN state s ON c.statecode = s.st_id
   WHERE LOWER(s.st_name) = 'lagos'
   -- or use state code
   WHERE s.st_abreviation = 'LAG'

3. Gender Queries:
   WHERE UPPER(gender) = 'M' for males
   WHERE UPPER(gender) = 'F' for females

4. Score Analysis:
   - Use candidate_scores for subject-specific scores
   - Use aggregate for overall performance
   - Consider year in analysis

Important Notes:
1. ALWAYS return a complete SQL query starting with SELECT
2. Use appropriate JOINs when combining tables
3. Include WHERE clauses for data quality
4. For aggregations, use GROUP BY and HAVING appropriately
5. Format numbers using ROUND() for better readability
6. Limit results to a reasonable number (e.g., LIMIT 20)
7. Use COUNT(*) for counting records
8. Always alias tables and complex columns
9. Consider NULL values in calculations

Return ONLY the SQL query, no explanations.`, query)

	resp, err := chat.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return fmt.Errorf("error sending message to Gemini: %v", err)
	}

	// Extract SQL query from response
	sqlQuery := e.extractSQLFromResponse(resp)
	if sqlQuery == "" {
		return fmt.Errorf("no valid SQL query generated - try rephrasing your question to be more specific about what you want to know about candidates, courses, or scores")
	}

	fmt.Printf("\nGenerated SQL Query:\n%s\n\n", sqlQuery)

	// Execute the query
	results, err := e.executeQuery(sqlQuery)
	if err != nil {
		return fmt.Errorf("error executing query: %v\n\nTry rephrasing your question or being more specific about what information you need", err)
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
				// Clean up the response
				response := string(text)
				response = strings.TrimSpace(response)

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

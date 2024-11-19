package nlquery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/nonsonwune/spk2_db/nlquery/prompts"
	"google.golang.org/api/option"
)

type NLQueryEngine struct {
	client        *genai.Client
	model         *genai.GenerativeModel
	db            *sql.DB
	promptBuilder *prompts.PromptBuilder
	keyManager    *KeyManager
}

type QueryResult struct {
	ThoughtProcess string
	SQLQuery      string
	Explanation   string
	Results       string
}

func NewNLQueryEngine(db *sql.DB) (*NLQueryEngine, error) {
	keyManager := NewKeyManager()
	if len(keyManager.keys) == 0 {
		return nil, fmt.Errorf("no API keys available")
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(keyManager.GetNextKey()))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.2)

	return &NLQueryEngine{
		client:        client,
		model:         model,
		db:            db,
		promptBuilder: prompts.NewPromptBuilder(),
		keyManager:    keyManager,
	}, nil
}

func (e *NLQueryEngine) generateWithRetry(ctx context.Context, prompt string) (string, error) {
	var lastErr error
	maxRetries := 3
	baseDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			fmt.Printf("\nRetrying API call (attempt %d/%d)...\n", attempt, maxRetries)
			
			// Get a new API key for the retry
			key := e.keyManager.GetNextKey()
			client, err := genai.NewClient(ctx, option.WithAPIKey(key))
			if err != nil {
				continue
			}
			e.client = client
			e.model = client.GenerativeModel("gemini-1.5-flash")
			e.model.SetTemperature(0.2)
		}

		// Create a context with timeout for this attempt
		timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		resp, err := e.model.GenerateContent(timeoutCtx, genai.Text(prompt))
		if err != nil {
			lastErr = err
			// Mark the current key as failed and try the next one
			e.keyManager.MarkKeyFailed("")
			time.Sleep(baseDelay * time.Duration(attempt))
			continue
		}

		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
				return string(text), nil
			}
		}
		lastErr = fmt.Errorf("unexpected response type")
		time.Sleep(baseDelay * time.Duration(attempt))
	}
	return "", fmt.Errorf("all retries failed: %v", lastErr)
}

func cleanJSONResponse(resp string) string {
    // Remove any markdown formatting
    resp = strings.ReplaceAll(resp, "```json", "")
    resp = strings.ReplaceAll(resp, "```", "")
    resp = strings.ReplaceAll(resp, "`", "")
    // Trim any whitespace before and after
    resp = strings.TrimSpace(resp)
    return resp
}

func cleanSQLQuery(sql string) string {
    // Replace escaped newlines with spaces
    sql = strings.ReplaceAll(sql, "\\n", " ")
    // Remove any extra whitespace
    sql = strings.Join(strings.Fields(sql), " ")
    return sql
}

func extractSQLFromResponse(resp string) (string, error) {
    // Clean the response first
    resp = cleanJSONResponse(resp)
    
    // Try to parse as JSON
    var result struct {
        SQLQuery string `json:"sql_query"`
    }
    
    if err := json.Unmarshal([]byte(resp), &result); err != nil {
        // If JSON parsing fails, try to extract SQL directly using regex
        sqlPattern := `(?i)SELECT\s+.*?(?:;|$)`
        re := regexp.MustCompile(sqlPattern)
        if matches := re.FindString(resp); matches != "" {
            return cleanSQLQuery(matches), nil
        }
        return "", fmt.Errorf("failed to extract SQL query: %v\nResponse was: %s", err, resp)
    }
    
    if result.SQLQuery == "" {
        return "", fmt.Errorf("no SQL query found in response")
    }
    
    return cleanSQLQuery(result.SQLQuery), nil
}

func (e *NLQueryEngine) cleanSQLResponse(sql string) string {
	// Remove markdown code block markers
	sql = strings.TrimPrefix(sql, "```sql")
	sql = strings.TrimPrefix(sql, "```")
	sql = strings.TrimSuffix(sql, "```")
	
	// Remove any leading/trailing whitespace
	sql = strings.TrimSpace(sql)
	
	return sql
}

func (e *NLQueryEngine) ProcessQuery(query string) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
    defer cancel()

    fmt.Println("\nAnalyzing query...")
    
    // Generate SQL query with retry
    prompt := e.promptBuilder.BuildQueryPrompt(query)
    resp, err := e.generateWithRetry(ctx, prompt)
    if err != nil {
        return "", fmt.Errorf("failed to generate SQL: %v", err)
    }

    // Extract and display thought process if available
    if strings.Contains(resp, "thought_process") {
        var result struct {
            ThoughtProcess string `json:"thought_process"`
        }
        cleanResp := cleanJSONResponse(resp)
        if err := json.Unmarshal([]byte(cleanResp), &result); err == nil && result.ThoughtProcess != "" {
            fmt.Printf("\nThought Process:\n%s\n", result.ThoughtProcess)
        }
    }

    // Extract SQL query
    sql, err := extractSQLFromResponse(resp)
    if err != nil {
        return "", fmt.Errorf("failed to extract SQL: %v\nResponse was: %s", err, resp)
    }

    fmt.Printf("\nGenerated SQL:\n%s\n", sql)

    fmt.Println("\nValidating query...")
    
    // Validate the generated SQL with retry
    validationPrompt := e.promptBuilder.BuildValidationPrompt(query, sql)
    validation, err := e.generateWithRetry(ctx, validationPrompt)
    if err != nil {
        return "", fmt.Errorf("failed to validate SQL: %v", err)
    }

    validation = strings.TrimSpace(validation)
    if !strings.EqualFold(validation, "VALID") {
        return "", fmt.Errorf("invalid SQL generated: %s", validation)
    }

    fmt.Println("\nExecuting query...")
    
    // Execute the SQL query
    rows, err := e.db.QueryContext(ctx, sql)
    if err != nil {
        // Generate user-friendly error message with retry
        errorPrompt := e.promptBuilder.BuildErrorPrompt(query, err)
        errorMsg, genErr := e.generateWithRetry(ctx, errorPrompt)
        if genErr == nil {
            return "", fmt.Errorf(errorMsg)
        }
        return "", fmt.Errorf("query failed: %v", err)
    }
    defer rows.Close()

    fmt.Println("\nFormatting results...")
    
    // Format results
    results, err := formatResults(rows)
    if err != nil {
        return "", fmt.Errorf("failed to format results: %v", err)
    }

    return results, nil
}

func formatResults(rows *sql.Rows) (string, error) {
    // Get column names
    columns, err := rows.Columns()
    if err != nil {
        return "", fmt.Errorf("failed to get column names: %v", err)
    }

    // Prepare values holder
    values := make([]interface{}, len(columns))
    valuePtrs := make([]interface{}, len(columns))
    for i := range columns {
        valuePtrs[i] = &values[i]
    }

    // Build result string
    var result strings.Builder
    
    // Write header
    maxWidths := make([]int, len(columns))
    for i, col := range columns {
        if i > 0 {
            result.WriteString("\t")
        }
        result.WriteString(col)
        maxWidths[i] = len(col)
    }
    result.WriteString("\n")
    
    // Write separator
    for i := range columns {
        if i > 0 {
            result.WriteString("\t")
        }
        result.WriteString(strings.Repeat("-", maxWidths[i]))
    }
    result.WriteString("\n")

    // Collect all rows first to calculate column widths
    var allRows [][]string
    for rows.Next() {
        err = rows.Scan(valuePtrs...)
        if err != nil {
            return "", fmt.Errorf("failed to scan row: %v", err)
        }

        // Convert row to strings
        rowStrings := make([]string, len(columns))
        for i, val := range values {
            if val == nil {
                rowStrings[i] = "NULL"
            } else {
                switch v := val.(type) {
                case []byte:
                    rowStrings[i] = string(v)
                default:
                    rowStrings[i] = fmt.Sprintf("%v", v)
                }
            }
            if len(rowStrings[i]) > maxWidths[i] {
                maxWidths[i] = len(rowStrings[i])
            }
        }
        allRows = append(allRows, rowStrings)
    }

    if err = rows.Err(); err != nil {
        return "", fmt.Errorf("error iterating rows: %v", err)
    }

    // Write rows with proper padding
    for _, rowStrings := range allRows {
        for i, val := range rowStrings {
            if i > 0 {
                result.WriteString("\t")
            }
            result.WriteString(val)
        }
        result.WriteString("\n")
    }

    if len(allRows) == 0 {
        result.WriteString("No results found\n")
    } else {
        result.WriteString(fmt.Sprintf("\nTotal rows: %d\n", len(allRows)))
    }

    return result.String(), nil
}

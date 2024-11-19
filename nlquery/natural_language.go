package nlquery

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/nonsonwune/spk2_db/nlquery/prompts"
)

type NLQueryEngine struct {
	db            *sql.DB
	model         *genai.GenerativeModel
	promptBuilder *prompts.PromptBuilder
}

func NewNLQueryEngine(db *sql.DB, model *genai.GenerativeModel) *NLQueryEngine {
	return &NLQueryEngine{
		db:            db,
		model:         model,
		promptBuilder: prompts.NewPromptBuilder(),
	}
}

func (e *NLQueryEngine) ProcessQuery(ctx context.Context, query string) (string, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	// Generate SQL query
	prompt := e.promptBuilder.BuildQueryPrompt(query)
	resp, err := e.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate SQL: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	sql := ""
	if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		sql = string(textPart)
	} else {
		return "", fmt.Errorf("unexpected response type")
	}
	sql = strings.TrimSpace(sql)

	// Validate the generated SQL
	validationPrompt := e.promptBuilder.BuildValidationPrompt(query, sql)
	validationResp, err := e.model.GenerateContent(ctx, genai.Text(validationPrompt))
	if err != nil {
		return "", fmt.Errorf("failed to validate SQL: %v", err)
	}

	if len(validationResp.Candidates) == 0 || len(validationResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no validation response generated")
	}

	validation := ""
	if textPart, ok := validationResp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		validation = string(textPart)
	} else {
		return "", fmt.Errorf("unexpected validation response type")
	}

	if !strings.HasPrefix(validation, "VALID") {
		return "", fmt.Errorf("invalid SQL generated: %s", validation)
	}

	// Execute the SQL query
	rows, err := e.db.QueryContext(ctx, sql)
	if err != nil {
		// Generate user-friendly error message
		errorPrompt := e.promptBuilder.BuildErrorPrompt(query, err)
		errorResp, genErr := e.model.GenerateContent(ctx, genai.Text(errorPrompt))
		if genErr != nil {
			return "", fmt.Errorf("query failed: %v", err)
		}
		if len(errorResp.Candidates) > 0 && len(errorResp.Candidates[0].Content.Parts) > 0 {
			if textPart, ok := errorResp.Candidates[0].Content.Parts[0].(genai.Text); ok {
				return "", fmt.Errorf(string(textPart))
			}
		}
		return "", fmt.Errorf("query failed: %v", err)
	}
	defer rows.Close()

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

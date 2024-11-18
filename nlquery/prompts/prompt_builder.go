package prompts

import (
	"fmt"
	"strings"
)

// PromptBuilder handles the construction of prompts for the LLM
type PromptBuilder struct {
	baseContext string
	examples    string
}

// NewPromptBuilder creates a new PromptBuilder with schema context
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		baseContext: SchemaContext,
		examples:    QueryExamples,
	}
}

// BuildQueryPrompt creates a prompt for SQL query generation
func (pb *PromptBuilder) BuildQueryPrompt(query string) string {
	return fmt.Sprintf(`You are an expert SQL query generator for a JAMB database.
Use this database schema and context to help you:

%s

Here are some example queries to learn from:
%s

Generate a PostgreSQL query for this question:
"%s"

Important:
1. Only return the SQL query, no explanations
2. Use proper table aliases (c for candidate, i for institution, etc.)
3. Always use COUNT(DISTINCT) for counting
4. Include appropriate JOINs based on relationships
5. Handle NULL values appropriately
6. Consider performance with large datasets

SQL Query:`, pb.baseContext, pb.examples, query)
}

// BuildValidationPrompt creates a prompt for validating generated SQL
func (pb *PromptBuilder) BuildValidationPrompt(query, sql string) string {
	return fmt.Sprintf(`You are a SQL query validator. Your task is to validate if the generated SQL query correctly answers the user's question.
Rules:
1. For medicine-related queries, check if the query handles variations (MEDICINE, MEDICAL, SURGERY)
2. For counting queries, verify proper use of COUNT and GROUP BY
3. Check table relationships and joins
4. Verify WHERE clause conditions match the question

User Question: %s
Generated SQL: %s

Respond with:
- "VALID" if the query is correct
- "INVALID: <reason>" if the query is incorrect, explaining why
`, query, sql)
}

// ExtractYear attempts to extract year from the query
func (pb *PromptBuilder) ExtractYear(query string) string {
	return fmt.Sprintf(`Extract the year from this query: "%s"

Rules:
1. Return only the 4-digit year if found
2. Return "current_year" if no specific year is mentioned
3. Handle variations like "in 2019", "during 2019", "2019 admissions"
4. If multiple years are mentioned, return the most relevant one

Year:`, strings.ToLower(query))
}

// BuildErrorPrompt creates a prompt for generating user-friendly error messages
func (pb *PromptBuilder) BuildErrorPrompt(query string, err error) string {
	return fmt.Sprintf(`Generate a user-friendly error message for this failed query:

Question: "%s"

Error: %v

Requirements:
1. Explain the issue in simple terms
2. Suggest how to rephrase the question
3. Keep the message concise and helpful

Error Message:`, query, err)
}

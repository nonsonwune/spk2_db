package prompts

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// QueryAgent defines the interface for query processing agents
type QueryAgent interface {
	Process(input string) (string, error)
}

// IntentAgent handles query intent recognition
type IntentAgent struct{}

func (a *IntentAgent) Process(query string) (string, error) {
	query = strings.ToLower(query)
	if strings.Contains(query, "highest") {
		return "highest_score", nil
	}
	if strings.Contains(query, "admitted") || strings.Contains(query, "admission") {
		return "admission_stats", nil
	}
	if strings.Contains(query, "registered") || strings.Contains(query, "registration") {
		return "registration_stats", nil
	}
	if strings.Contains(query, "disability") || strings.Contains(query, "disabilities") {
		return "disability_stats", nil
	}
	if strings.Contains(query, "score") || strings.Contains(query, "subject") {
		return "subject_scores", nil
	}
	if strings.Contains(query, "local") || strings.Contains(query, "lga") {
		return "lga_stats", nil
	}
	if strings.Contains(query, "faculty") || strings.Contains(query, "department") {
		return "faculty_stats", nil
	}
	if strings.Contains(query, "institution") || strings.Contains(query, "university") || strings.Contains(query, "school") {
		return "institution_stats", nil
	}
	return "general_stats", nil
}

// SchemaAgent handles database schema mapping
type SchemaAgent struct {
	schemaContext string
}

func (a *SchemaAgent) Process(query string) (string, error) {
	// Map query terms to database columns and tables
	tables := map[string][]string{
		"candidate": {
			"regnumber", "firstname", "surname", "gender", "statecode", 
			"aggregate", "year", "is_admitted", "is_direct_entry", 
			"maritalstatus", "is_blind", "is_deaf", "noofsittings",
			"app_course1", "inid", "lg_id", "date_of_birth",
			"is_mock_candidate", "malpractice",
		},
		"state": {
			"st_id", "st_name", "st_abreviation", "st_elds",
		},
		"course": {
			"course_code", "course_name", "course_abbreviation",
			"facid", "duration", "degree",
		},
		"institution": {
			"inid", "inabv", "inname", "inst_state_id",
			"affiliated_state_id", "intyp", "inst_cat",
		},
		"institution_type": {
			"intyp_id", "intyp_name",
		},
		"faculty": {
			"fac_id", "fac_name", "fac_code",
		},
		"lga": {
			"lg_id", "lg_st_id", "lg_name", "lg_abreviation",
		},
		"candidate_scores": {
			"cand_reg_number", "subject_id", "score",
		},
		"subject": {
			"subject_id", "subject_name",
		},
		"candidate_disabilities": {
			"cand_reg_number", "disability_type", "disability_level",
		},
	}
	
	joins := map[string]string{
		"state": "JOIN state s ON c.statecode = s.st_id",
		"course": "JOIN course co ON c.app_course1 = co.course_code",
		"institution": "JOIN institution i ON c.inid = i.inid",
		"institution_type": "JOIN institution_type it ON i.intyp = it.intyp_id",
		"faculty": "JOIN faculty f ON co.facid = f.fac_id",
		"lga": "JOIN lga l ON c.lg_id = l.lg_id",
		"candidate_scores": "LEFT JOIN candidate_scores cs ON c.regnumber = cs.cand_reg_number",
		"subject": "LEFT JOIN subject sub ON cs.subject_id = sub.subject_id",
		"candidate_disabilities": "LEFT JOIN candidate_disabilities cd ON c.regnumber = cd.cand_reg_number",
	}
	
	return fmt.Sprintf("%v|%v", tables, joins), nil
}

// PromptBuilder handles the construction of prompts for the LLM
type PromptBuilder struct {
	baseContext    string
	examples       string
	courseMatcher  *CourseNameMatcher
	intentAgent    *IntentAgent
	schemaAgent    *SchemaAgent
}

// NewPromptBuilder creates a new PromptBuilder with schema context
func NewPromptBuilder() *PromptBuilder {
	// Get the current file's directory
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	
	matcher := NewCourseNameMatcher()
	err := matcher.LoadCourseNames(filepath.Join(dir, "course_names.txt"))
	if err != nil {
		fmt.Printf("Warning: Failed to load course names: %v\n", err)
	}

	return &PromptBuilder{
		baseContext:    SchemaContext,
		examples:       QueryExamples,
		courseMatcher:  matcher,
		intentAgent:    &IntentAgent{},
		schemaAgent:    &SchemaAgent{},
	}
}

// BuildQueryPrompt creates a prompt for SQL query generation
func (pb *PromptBuilder) BuildQueryPrompt(query string) string {
	// Process query through agents
	intent, _ := pb.intentAgent.Process(query)
	schema, _ := pb.schemaAgent.Process(query)
	
	return fmt.Sprintf(`
Given the following database schema:
%s

And the query intent: %s
With schema mapping: %s

Generate a SQL query that:
1. Correctly joins necessary tables
2. Uses proper column names from the schema
3. Handles aggregations and grouping appropriately
4. Includes appropriate indexes and constraints
5. Considers performance implications

User Query: %s
`, pb.baseContext, intent, schema, query)
}

// BuildValidationPrompt creates a prompt for validating generated SQL
func (pb *PromptBuilder) BuildValidationPrompt(query, sql string) string {
	return fmt.Sprintf(`You are a SQL query validator. Your task is to validate if the generated SQL query correctly answers the user's question.

User Query: %s

Generated SQL:
%s

Please validate:
1. Table joins are correct and necessary
2. Column names match the schema
3. WHERE conditions are appropriate
4. GROUP BY and ORDER BY are logical
5. Performance considerations are addressed

Provide feedback on:
1. Correctness
2. Performance
3. Suggested improvements`, query, sql)
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

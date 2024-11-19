package prompts

import (
	"fmt"
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
type PromptBuilder struct{}

func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

func (pb *PromptBuilder) BuildQueryPrompt(query string) string {
	return fmt.Sprintf(`You are a SQL query generator for a JAMB (Joint Admissions and Matriculation Board) database.

%s

%s

Given this context, generate a SQL query for this question:
%s

Return only the SQL query without any explanation or markdown formatting.`, SchemaContext, QueryExamples, query)
}

func (pb *PromptBuilder) BuildErrorPrompt(query string, err error) string {
	return fmt.Sprintf(`Given this failed query attempt:
Query: %s
Error: %s

Explain in simple terms what went wrong and how the user can modify their question to get better results.
Be specific about what terms or filters they could add to make the query more precise.`, query, err)
}

func (pb *PromptBuilder) BuildValidationPrompt(query string, sql string) string {
	return fmt.Sprintf(`Validate this SQL query for the JAMB database:
Original Question: %s
Generated SQL:
%s

Check for:
1. SQL syntax errors
2. Missing or incorrect table joins
3. Appropriate filtering conditions
4. Proper grouping and ordering
5. Reasonable result limits

Respond with either:
"VALID" if the query looks correct
or
"INVALID: [reason]" if there are issues`, query, sql)
}

func (pb *PromptBuilder) ExtractYear(query string) string {
	return fmt.Sprintf(`Extract the year from this query: "%s"

Rules:
1. Return only the 4-digit year if found
2. Return "current_year" if no specific year is mentioned
3. Handle variations like "in 2019", "during 2019", "2019 admissions"
4. If multiple years are mentioned, return the most relevant one

Year:`, strings.ToLower(query))
}

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
	
	a.schemaContext = fmt.Sprintf("%v|%v", tables, joins)
	return a.schemaContext, nil
}

// PromptBuilder handles the construction of prompts for the LLM
type PromptBuilder struct {
    schemaContext string
}

func NewPromptBuilder() *PromptBuilder {
    schemaAgent := &SchemaAgent{}
    schemaContext, _ := schemaAgent.Process("")
    return &PromptBuilder{
        schemaContext: schemaContext,
    }
}

func (pb *PromptBuilder) BuildQueryPrompt(query string) string {
    return fmt.Sprintf(`You are a SQL query generator for a JAMB database system. Your task is to convert natural language questions into SQL queries.

Database Schema:
%s

User Question: %s

Instructions:
1. Analyze the question carefully
2. Consider the database schema
3. Generate a valid PostgreSQL query
4. Return your response in this exact JSON format:
{
    "thought_process": "Step by step explanation of your reasoning",
    "sql_query": "The complete SQL query with proper table aliases and joins",
    "explanation": "Brief explanation of what the query does"
}

Important Rules:
1. Use proper table aliases (e.g., c for candidate, co for course)
2. Always check if tables exist before joining
3. Use INNER JOIN for required relationships, LEFT JOIN for optional ones
4. Double check column names match the schema exactly
5. For course name matching:
   - Exact single course (e.g., "Pharmacy"): 
     UPPER(co.course_name) = 'PHARMACY'
   - Related courses (e.g., "pharmacy-related", "pharmacy courses"):
     LOWER(co.course_name) LIKE LOWER('%%pharm%%')
   - Multiple specific courses:
     UPPER(co.course_name) IN ('MEDICINE', 'SURGERY', 'DENTISTRY')
6. Return ONLY the JSON response with NO markdown formatting

Course Query Guidelines:
- Single course: "who applied pharmacy"
  → UPPER(co.course_name) = 'PHARMACY'
  
- Related courses: "who applied pharmacy courses"
  → LOWER(co.course_name) LIKE LOWER('%%pharm%%')
  
- Multiple courses: "who applied medicine or surgery"
  → UPPER(co.course_name) IN ('MEDICINE', 'SURGERY')
  
- Course field: "who applied medical courses"
  → LOWER(co.course_name) LIKE LOWER('%%medic%%')
  OR LOWER(co.course_name) LIKE LOWER('%%surge%%')
  OR LOWER(co.course_name) LIKE LOWER('%%doctor%%')

Example Responses:
{
    "thought_process": "1. User wants specific course statistics\n2. Use exact match\n3. Join and filter",
    "sql_query": "SELECT c.gender, COUNT(*) AS applicant_count FROM candidate c INNER JOIN course co ON c.app_course1 = co.course_code WHERE c.year = 2023 AND UPPER(co.course_name) = 'MEDICINE' GROUP BY c.gender",
    "explanation": "Counts male and female candidates who applied for Medicine specifically"
}

{
    "thought_process": "1. User wants medical field statistics\n2. Use pattern matching\n3. Join and filter",
    "sql_query": "SELECT c.gender, COUNT(*) AS applicant_count FROM candidate c INNER JOIN course co ON c.app_course1 = co.course_code WHERE c.year = 2023 AND (LOWER(co.course_name) LIKE LOWER('%%medic%%') OR LOWER(co.course_name) LIKE LOWER('%%surge%%')) GROUP BY c.gender",
    "explanation": "Counts male and female candidates who applied for any medical-related course"
}`, pb.schemaContext, query)
}

func (pb *PromptBuilder) BuildErrorPrompt(query string, err error) string {
    return fmt.Sprintf(`Given this error while executing a SQL query: %v

Please explain what went wrong in user-friendly terms, considering this was the original question:
%s

Return ONLY the explanation with NO markdown formatting or code blocks.`, err, query)
}

func (pb *PromptBuilder) BuildValidationPrompt(query, sql string) string {
    return fmt.Sprintf(`Validate this SQL query for the JAMB database:

Original Question: %s

Generated SQL:
%s

Database Schema:
%s

Return "VALID" if the query is correct, or explain the specific issues if invalid. Check for:
1. Correct table and column names
2. Proper JOIN conditions
3. Correct filtering conditions
4. Appropriate GROUP BY if using aggregations
5. No syntax errors

Return ONLY "VALID" or a specific error message.`, query, sql, pb.schemaContext)
}

func (pb *PromptBuilder) ExtractYear(query string) string {
    query = strings.ToLower(query)
    if strings.Contains(query, "2020") {
        return "2020"
    }
    if strings.Contains(query, "2021") {
        return "2021"
    }
    if strings.Contains(query, "2022") {
        return "2022"
    }
    if strings.Contains(query, "2023") {
        return "2023"
    }
    return "2023" // Default to latest year
}

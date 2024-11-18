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
	return fmt.Sprintf(`You are a SQL query generator for a JAMB database. Follow these rules strictly:

1. Database Statistics:
   - Total courses: 20,399
   - Actual named courses: 17,362
   - Placeholder courses: 3,037 (format: "Course XXXXX")

2. Table Structure:
   - candidate: Main table with regnumber, gender, year, app_course1, statecode
   - course: Contains course_code, course_name
   - state: Contains st_id, st_name

3. Course Categories:
   A. Science & Engineering:
      - Engineering courses (Aerospace, Agricultural, Biomedical, etc.)
      - Pure Sciences (Physics, Chemistry, Biology)
      - Applied Sciences (Applied Mathematics, Statistics)
   
   B. Medical & Health Sciences:
      - Medicine & Surgery
      - Veterinary Medicine
      - Medical Laboratory Science
      - Medical Biochemistry
      - Public Health
   
   C. Arts & Humanities:
      - Languages (Arabic, English, French, etc.)
      - Religious Studies
      - History & Archaeology
   
   D. Social Sciences & Management:
      - Accounting & Finance
      - Economics
      - Business Administration
   
   E. Agriculture & Environmental Sciences:
      - Agricultural Science
      - Animal Science
      - Environmental Management

4. Query Rules:
   - Always exclude placeholder courses:
     AND LOWER(co.course_name) NOT LIKE 'course%%'
   - For medicine/health queries use:
     LOWER(co.course_name) LIKE ANY(ARRAY['%%medicine%%', '%%medical%%', '%%health%%', '%%pharm%%', '%%surg%%'])
   - Always use UPPER() for gender: UPPER(c.gender) = 'M' or 'F'
   - Use LOWER() for state names: LOWER(s.st_name) = LOWER('state_name')

5. Best Practices:
   - Use table aliases: candidate AS c, course AS co, state AS s
   - Always use proper JOIN conditions
   - Use COUNT(DISTINCT c.regnumber) for counting candidates
   - Include year in WHERE clause when relevant

Example Queries:
1. "how many women from anambra applied for medicine in 2023"
   SELECT COUNT(DISTINCT c.regnumber)
   FROM candidate c
   JOIN course co ON c.app_course1 = co.course_code
   JOIN state s ON c.statecode = s.st_id
   WHERE UPPER(c.gender) = 'F'
   AND LOWER(s.st_name) = LOWER('anambra')
   AND LOWER(co.course_name) LIKE ANY(ARRAY['%%medicine%%', '%%medical%%', '%%surg%%'])
   AND LOWER(co.course_name) NOT LIKE 'course%%'
   AND c.year = 2023;

2. "count male candidates in 2023"
   SELECT COUNT(DISTINCT c.regnumber)
   FROM candidate c
   WHERE UPPER(c.gender) = 'M'
   AND c.year = 2023;

Now generate a SQL query for this question: %s`, query)
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

package prompts

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// PromptBuilder handles the construction of prompts for the LLM
type PromptBuilder struct {
	baseContext    string
	examples       string
	courseMatcher *CourseNameMatcher
}

// NewPromptBuilder creates a new PromptBuilder with schema context
func NewPromptBuilder() *PromptBuilder {
	// Get the current file's directory
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	
	matcher := NewCourseNameMatcher()
	err := matcher.LoadCourseNames(filepath.Join(dir, "course_names.txt"))
	if err != nil {
		// Log error but continue - we'll fall back to basic matching
		fmt.Printf("Warning: Failed to load course names: %v\n", err)
	}

	return &PromptBuilder{
		baseContext:    SchemaContext,
		examples:       QueryExamples,
		courseMatcher: matcher,
	}
}

// BuildQueryPrompt creates a prompt for SQL query generation
func (pb *PromptBuilder) BuildQueryPrompt(query string) string {
	// Get course patterns
	var coursePatterns []string
	if pb.courseMatcher != nil {
		coursePatterns = pb.courseMatcher.FindMatchingCourses(query)
	}

	// Build course pattern string
	var coursePattern string
	if len(coursePatterns) > 0 {
		coursePattern = "LIKE ANY(ARRAY[" + strings.Join(coursePatterns, ", ") + "])"
	} else {
		coursePattern = "NOT LIKE 'Course %'"
	}

	return fmt.Sprintf(`You are a SQL query generator for a JAMB database. Follow these rules strictly:

1. Database Structure:
   - candidate: Contains application data (regnumber, gender, year, app_course1, statecode)
   - course: Course information (course_code, course_name)
   - institution: School information (inid, inname)
   - state: State information (st_id, st_name)

2. Course Name Matching:
   For this query, use the following course pattern:
   WHERE co.course_name %s

3. Query Best Practices:
   - Always use table aliases: 
     candidate AS c
     course AS co
     institution AS i
     state AS s
   - Count distinct candidates: COUNT(DISTINCT c.regnumber)
   - For time trends, group by year: GROUP BY c.year ORDER BY c.year
   - For gender analysis: UPPER(c.gender) IN ('M', 'F')

Example Queries:

1. "which year had the most medicine applicants"
   SELECT c.year, COUNT(DISTINCT c.regnumber) as applicant_count
   FROM candidate c
   JOIN course co ON c.app_course1 = co.course_code
   WHERE co.course_name LIKE ANY(ARRAY['%%MEDICINE & SURGERY%%', '%%MEDICAL%%'])
   GROUP BY c.year
   ORDER BY applicant_count DESC
   LIMIT 1;

2. "how many people applied for engineering in 2023"
   SELECT COUNT(DISTINCT c.regnumber)
   FROM candidate c
   JOIN course co ON c.app_course1 = co.course_code
   WHERE co.course_name LIKE ANY(ARRAY['%%ENGINEERING%%'])
   AND c.year = 2023;

Now generate a SQL query for this question: %s`, coursePattern, query)
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

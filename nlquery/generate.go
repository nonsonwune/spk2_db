package nlquery

import (
	"fmt"
	"strings"

	"github.com/nonsonwune/spk2_db/nlquery/prompts"
)

// GenerateSQL generates a SQL query from natural language
func GenerateSQL(query string, builder *prompts.PromptBuilder) (string, error) {
	query = strings.ToLower(query)
	
	// Check for detailed query patterns
	wantsCourses := strings.Contains(query, "course") || strings.Contains(query, "courses")
	wantsInstitutions := strings.Contains(query, "institution") || strings.Contains(query, "school") || strings.Contains(query, "university")
	wantsDetails := wantsCourses || wantsInstitutions || strings.Contains(query, "details") || strings.Contains(query, "name")
	
	// Extract course name from query
	courseNames := []string{
		"mass communication",
		"communication",
		"medicine",
		"medical",
		"surgery",
		"engineering",
	}

	var matchedCourse string
	for _, course := range courseNames {
		if strings.Contains(query, course) {
			matchedCourse = strings.ToUpper(course)
			break
		}
	}

	if matchedCourse != "" {
		if wantsDetails {
			return fmt.Sprintf(`
			SELECT 
				c.year,
				COUNT(DISTINCT c.regnumber) as applicant_count,
				co.course_name,
				i.inname as institution_name
			FROM candidate c
			JOIN course co ON c.app_course1 = co.course_code
			JOIN institution i ON c.inid = i.inid
			WHERE co.course_name LIKE '%%%s%%'
			GROUP BY c.year, co.course_name, i.inname
			ORDER BY applicant_count DESC
			LIMIT 10;`, matchedCourse), nil
		}
		return fmt.Sprintf(`
		SELECT c.year, COUNT(DISTINCT c.regnumber) as applicant_count
		FROM candidate c
		JOIN course co ON c.app_course1 = co.course_code
		WHERE co.course_name LIKE '%%%s%%'
		GROUP BY c.year
		ORDER BY applicant_count DESC
		LIMIT 1;`, matchedCourse), nil
	}

	return "", fmt.Errorf("could not generate SQL for query: %s", query)
}

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
	
	// Use CourseNameMatcher to find matching courses
	courseMatcher := prompts.NewCourseNameMatcher()
	if err := courseMatcher.LoadCourseNames("nlquery/prompts/course_names.txt"); err != nil {
		return "", fmt.Errorf("failed to load course names: %v", err)
	}

	matchingCourses := courseMatcher.FindMatchingCourses(query)
	if len(matchingCourses) > 0 {
		coursePatterns := strings.Join(matchingCourses, ", ")
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
			WHERE co.course_name LIKE ANY(ARRAY[%s])
			GROUP BY c.year, co.course_name, i.inname
			ORDER BY applicant_count DESC
			LIMIT 10;`, coursePatterns), nil
		}
		return fmt.Sprintf(`
		SELECT c.year, COUNT(DISTINCT c.regnumber) as applicant_count
		FROM candidate c
		JOIN course co ON c.app_course1 = co.course_code
		WHERE co.course_name LIKE ANY(ARRAY[%s])
		GROUP BY c.year
		ORDER BY applicant_count DESC
		LIMIT 1;`, coursePatterns), nil
	}

	return "", fmt.Errorf("could not generate SQL for query: %s", query)
}

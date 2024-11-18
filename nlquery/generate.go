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
	
	// Mass Communication queries
	if strings.Contains(query, "mass communication") || strings.Contains(query, "communication") {
		if wantsDetails {
			return `
			SELECT 
				c.year,
				COUNT(DISTINCT c.regnumber) as applicant_count,
				co.course_name,
				i.inname as institution_name
			FROM candidate c
			JOIN course co ON c.app_course1 = co.course_code
			JOIN institution i ON c.inid = i.inid
			WHERE co.course_name LIKE ANY(ARRAY['%MASS COMMUNICATION%', '%COMMUNICATION%'])
			GROUP BY c.year, co.course_name, i.inname
			ORDER BY applicant_count DESC
			LIMIT 10;`, nil
		}
		return `
		SELECT c.year, COUNT(DISTINCT c.regnumber) as applicant_count
		FROM candidate c
		JOIN course co ON c.app_course1 = co.course_code
		WHERE co.course_name LIKE ANY(ARRAY['%MASS COMMUNICATION%', '%COMMUNICATION%'])
		GROUP BY c.year
		ORDER BY applicant_count DESC
		LIMIT 1;`, nil
	} else if strings.Contains(query, "medicine") || strings.Contains(query, "medical") {
		if wantsDetails {
			return `
			SELECT 
				c.year,
				COUNT(DISTINCT c.regnumber) as applicant_count,
				co.course_name,
				i.inname as institution_name
			FROM candidate c
			JOIN course co ON c.app_course1 = co.course_code
			JOIN institution i ON c.inid = i.inid
			WHERE co.course_name LIKE ANY(ARRAY['%MEDICINE%', '%MEDICAL%', '%SURGERY%'])
			GROUP BY c.year, co.course_name, i.inname
			ORDER BY applicant_count DESC
			LIMIT 10;`, nil
		}
		return `
		SELECT c.year, COUNT(DISTINCT c.regnumber) as applicant_count
		FROM candidate c
		JOIN course co ON c.app_course1 = co.course_code
		WHERE co.course_name LIKE ANY(ARRAY['%MEDICINE%', '%MEDICAL%', '%SURGERY%'])
		GROUP BY c.year
		ORDER BY applicant_count DESC
		LIMIT 1;`, nil
	} else if strings.Contains(query, "engineering") {
		if wantsDetails {
			return `
			SELECT 
				c.year,
				COUNT(DISTINCT c.regnumber) as applicant_count,
				co.course_name,
				i.inname as institution_name
			FROM candidate c
			JOIN course co ON c.app_course1 = co.course_code
			JOIN institution i ON c.inid = i.inid
			WHERE co.course_name LIKE '%ENGINEERING%'
			GROUP BY c.year, co.course_name, i.inname
			ORDER BY applicant_count DESC
			LIMIT 10;`, nil
		}
		return `
		SELECT c.year, COUNT(DISTINCT c.regnumber) as applicant_count
		FROM candidate c
		JOIN course co ON c.app_course1 = co.course_code
		WHERE co.course_name LIKE '%ENGINEERING%'
		GROUP BY c.year
		ORDER BY applicant_count DESC
		LIMIT 1;`, nil
	}

	return "", fmt.Errorf("could not generate SQL for query: %s", query)
}

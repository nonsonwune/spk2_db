package nlquery

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// GenerateSQL generates a SQL query from natural language
func GenerateSQL(query string) (string, string, error) {
	queryLower := strings.ToLower(query)
	
	// Extract key components from the query
	hasRegion := strings.Contains(queryLower, "region") || 
		strings.Contains(queryLower, "state") || 
		strings.Contains(queryLower, "location") ||
		strings.Contains(queryLower, "area")

	// For course-related queries, use direct pattern matching
	coursePattern := fmt.Sprintf("'%%%s%%'", strings.ReplaceAll(queryLower, "'", "''"))

	if hasRegion {
		description := "Analysis of applications by region"
		sqlQuery := fmt.Sprintf(`
			WITH RegionalStats AS (
				SELECT 
					CASE 
						WHEN s.st_name IN ('BENUE', 'FCT', 'KOGI', 'KWARA', 'NASARAWA', 'NIGER', 'PLATEAU') THEN 'North Central'
						WHEN s.st_name IN ('ADAMAWA', 'BAUCHI', 'BORNO', 'GOMBE', 'TARABA', 'YOBE') THEN 'North East'
						WHEN s.st_name IN ('JIGAWA', 'KADUNA', 'KANO', 'KATSINA', 'KEBBI', 'SOKOTO', 'ZAMFARA') THEN 'North West'
						WHEN s.st_name IN ('ABIA', 'ANAMBRA', 'EBONYI', 'ENUGU', 'IMO') THEN 'South East'
						WHEN s.st_name IN ('AKWA IBOM', 'BAYELSA', 'CROSS RIVER', 'DELTA', 'EDO', 'RIVERS') THEN 'South South'
						WHEN s.st_name IN ('EKITI', 'LAGOS', 'OGUN', 'ONDO', 'OSUN', 'OYO') THEN 'South West'
					END as region,
					s.st_name as state_name,
					co.course_name,
					COUNT(DISTINCT c.regnumber) as total_applicants,
					COUNT(DISTINCT CASE WHEN c.is_admitted = true THEN c.regnumber END) as admitted_count
				FROM candidate c
				JOIN state s ON c.statecode = s.st_id
				JOIN course co ON c.app_course1 = co.course_code
				WHERE LOWER(co.course_name) LIKE %s
				AND c.year = 2023
				GROUP BY 
					CASE 
						WHEN s.st_name IN ('BENUE', 'FCT', 'KOGI', 'KWARA', 'NASARAWA', 'NIGER', 'PLATEAU') THEN 'North Central'
						WHEN s.st_name IN ('ADAMAWA', 'BAUCHI', 'BORNO', 'GOMBE', 'TARABA', 'YOBE') THEN 'North East'
						WHEN s.st_name IN ('JIGAWA', 'KADUNA', 'KANO', 'KATSINA', 'KEBBI', 'SOKOTO', 'ZAMFARA') THEN 'North West'
						WHEN s.st_name IN ('ABIA', 'ANAMBRA', 'EBONYI', 'ENUGU', 'IMO') THEN 'South East'
						WHEN s.st_name IN ('AKWA IBOM', 'BAYELSA', 'CROSS RIVER', 'DELTA', 'EDO', 'RIVERS') THEN 'South South'
						WHEN s.st_name IN ('EKITI', 'LAGOS', 'OGUN', 'ONDO', 'OSUN', 'OYO') THEN 'South West'
					END,
					s.st_name,
					co.course_name
			)
			SELECT 
				region,
				state_name,
				course_name,
				total_applicants,
				admitted_count,
				ROUND(100.0 * total_applicants / SUM(total_applicants) OVER (), 2) as percentage_of_total,
				ROUND(100.0 * admitted_count / NULLIF(total_applicants, 0), 2) as admission_rate
			FROM RegionalStats
			ORDER BY total_applicants DESC;
		`, coursePattern)
		return sqlQuery, description, nil
	}

	// Default to a simple course analysis if no specific analysis type is detected
	description := "Analysis of course applications"
	sqlQuery := fmt.Sprintf(`
		SELECT 
			co.course_name,
			COUNT(DISTINCT c.regnumber) as total_applicants,
			COUNT(DISTINCT CASE WHEN c.is_admitted = true THEN c.regnumber END) as admitted_count,
			ROUND(100.0 * COUNT(DISTINCT CASE WHEN c.is_admitted = true THEN c.regnumber END)::numeric / 
				NULLIF(COUNT(DISTINCT c.regnumber), 0), 2) as admission_rate
		FROM candidate c
		JOIN course co ON c.app_course1 = co.course_code
		WHERE LOWER(co.course_name) LIKE %s
		AND c.year = 2023
		GROUP BY co.course_name
		ORDER BY total_applicants DESC
		LIMIT 20;
	`, coursePattern)
	
	return sqlQuery, description, nil
}

// FormatQueryResult formats the query results and saves them to a file
func FormatQueryResult(query string, sql string, description string, rows *sql.Rows) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("query_tables/query_result_%s.txt", timestamp)

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// Create the formatted output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Query: %s\n\n", query))
	output.WriteString("Generated SQL Query:\n\n")
	output.WriteString(sql + "\n\n")
	output.WriteString("Results Table:\n")
	output.WriteString("--------------\n\n")

	// Calculate column widths
	columnWidths := make([]int, len(columns))
	for i, col := range columns {
		columnWidths[i] = len(col)
	}

	// Prepare value holders
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	// Get all rows to calculate max column widths
	var allRows [][]string
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return err
		}

		row := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", val)
			}
			if len(row[i]) > columnWidths[i] {
				columnWidths[i] = len(row[i])
			}
		}
		allRows = append(allRows, row)
	}

	// Write column headers
	for i, col := range columns {
		format := fmt.Sprintf("%%-%ds", columnWidths[i]+2)
		output.WriteString(fmt.Sprintf(format, col))
		if i < len(columns)-1 {
			output.WriteString("| ")
		}
	}
	output.WriteString("\n")

	// Write separator line
	for i, width := range columnWidths {
		output.WriteString(strings.Repeat("-", width+2))
		if i < len(columnWidths)-1 {
			output.WriteString("+-")
		}
	}
	output.WriteString("\n")

	// Write data rows
	for _, row := range allRows {
		for i, val := range row {
			format := fmt.Sprintf("%%-%ds", columnWidths[i]+2)
			output.WriteString(fmt.Sprintf(format, val))
			if i < len(columns)-1 {
				output.WriteString("| ")
			}
		}
		output.WriteString("\n")
	}

	output.WriteString("\nTable Description: " + description + "\n")

	// Write to file
	err = os.WriteFile(filename, []byte(output.String()), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Query results saved to: %s\n", filename)
	return nil
}

func ExecuteAndFormatQuery(db *sql.DB, query string, sql string, description string) error {
	rows, err := db.Query(sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	return FormatQueryResult(query, sql, description, rows)
}

func extractState(query string) string {
	states := []string{"abia", "adamawa", "akwa ibom", "anambra", "bauchi", "bayelsa", "benue", "borno", "cross river", "delta", "ebonyi", "edo", "ekiti", "enugu", "gombe", "imo", "jigawa", "kaduna", "kano", "katsina", "kebbi", "kogi", "kwara", "lagos", "nasarawa", "niger", "ogun", "ondo", "osun", "oyo", "plateau", "rivers", "sokoto", "taraba", "yobe", "zamfara", "fct"}
	
	for _, state := range states {
		if strings.Contains(query, state) {
			return strings.Title(state)
		}
	}
	return ""
}

func extractYear(query string) string {
	re := regexp.MustCompile(`\b20\d{2}\b`)
	match := re.FindString(query)
	if match != "" {
		return match
	}
	return "2023" // Default to current year if no year specified
}

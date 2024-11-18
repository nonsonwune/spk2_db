package models

import "database/sql"

// HistoricalCourseCode represents the historical_course_codes table
type HistoricalCourseCode struct {
	ID            int            `db:"id" json:"id"`
	CourseCode    string         `db:"course_code" json:"course_code"`
	CourseName    string         `db:"course_name" json:"course_name"`
	InstitutionID string         `db:"institution_id" json:"institution_id"`
	Year          int            `db:"year" json:"year"`
	DateCreated   sql.NullTime   `db:"date_created" json:"date_created,omitempty"`
	Notes         sql.NullString `db:"notes" json:"notes,omitempty"`
}

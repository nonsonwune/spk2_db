package models

import "time"

// Course represents the course table
type Course struct {
	CourseCode     string    `db:"course_code" json:"course_code"`
	CourseName     string    `db:"course_name" json:"course_name"`
	Abbreviation   string    `db:"course_abbreviation" json:"abbreviation"`
	FacultyID      int       `db:"faculty_id" json:"faculty_id"`
	Duration       int       `db:"duration" json:"duration"`
	Degree         string    `db:"degree" json:"degree"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`

	// Relationships
	Faculty        *Faculty  `db:"-" json:"faculty,omitempty"`
}

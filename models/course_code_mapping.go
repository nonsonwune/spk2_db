package models

import "database/sql"

// CourseCodeMapping represents the course_code_mappings table
type CourseCodeMapping struct {
	ID              int            `db:"id" json:"id"`
	OldCourseCode   string         `db:"old_course_code" json:"old_course_code"`
	NewCourseCode   string         `db:"new_course_code" json:"new_course_code"`
	InstitutionID   string         `db:"institution_id" json:"institution_id"`
	EffectiveFrom   sql.NullTime   `db:"effective_from" json:"effective_from,omitempty"`
	EffectiveTo     sql.NullTime   `db:"effective_to" json:"effective_to,omitempty"`
	MappingReason   sql.NullString `db:"mapping_reason" json:"mapping_reason,omitempty"`
	DateCreated     sql.NullTime   `db:"date_created" json:"date_created,omitempty"`
}

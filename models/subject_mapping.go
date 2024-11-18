package models

import "database/sql"

// SubjectMapping2023 represents the subject_mapping_2023 table
type SubjectMapping2023 struct {
	ID          int            `db:"id" json:"id"`
	OldSubjID   int            `db:"old_subj_id" json:"old_subj_id"`
	NewSubjID   int            `db:"new_subj_id" json:"new_subj_id"`
	DateCreated sql.NullTime   `db:"date_created" json:"date_created,omitempty"`
	Notes       sql.NullString `db:"notes" json:"notes,omitempty"`
}

package models

import "time"

// CandidateScore represents a candidate's score in a subject
type CandidateScore struct {
	CandRegNumber string    `db:"cand_reg_number" json:"cand_reg_number"`
	SubjectID     int       `db:"subject_id" json:"subject_id"`
	Score         int       `db:"score" json:"score"`
	Year          int       `db:"year" json:"year"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
	Subject       *Subject  `db:"-" json:"subject,omitempty"`
}

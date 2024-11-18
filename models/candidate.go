package models

import (
	"database/sql"
	"time"
)

// Candidate represents the candidates table
type Candidate struct {
	RegNumber     string         `db:"reg_number" json:"reg_number"`
	Year          int            `db:"year" json:"year"`
	MaritalStatus sql.NullString `db:"marital_status" json:"marital_status,omitempty"`
	Address       sql.NullString `db:"address" json:"address,omitempty"`
	Email         sql.NullString `db:"email" json:"email,omitempty"`
	GSMNo         sql.NullString `db:"gsm_no" json:"gsm_no,omitempty"`
	Surname       sql.NullString `db:"surname" json:"surname,omitempty"`
	FirstName     sql.NullString `db:"first_name" json:"first_name,omitempty"`
	MiddleName    sql.NullString `db:"middle_name" json:"middle_name,omitempty"`
	DateOfBirth   time.Time      `db:"date_of_birth" json:"date_of_birth"`
	Gender        sql.NullString `db:"gender" json:"gender,omitempty"`
	StateCode     sql.NullInt64  `db:"state_code" json:"state_code,omitempty"`
	LGID          sql.NullInt64  `db:"lg_id" json:"lg_id,omitempty"`
	IsAdmitted    sql.NullBool   `db:"is_admitted" json:"is_admitted,omitempty"`
	IsDirectEntry sql.NullBool   `db:"is_direct_entry" json:"is_direct_entry,omitempty"`
	Malpractice   sql.NullString `db:"malpractice" json:"malpractice,omitempty"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`

	// Relationships
	State           *State                 `db:"-" json:"state,omitempty"`
	LGA             *LGA                   `db:"-" json:"lga,omitempty"`
	Scores          []CandidateScore       `db:"-" json:"scores,omitempty"`
	Disabilities    *CandidateDisabilities `db:"-" json:"disabilities,omitempty"`
	ExamInfo        *CandidateExamInfo     `db:"-" json:"exam_info,omitempty"`
}

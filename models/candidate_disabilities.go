package models

import "time"

// CandidateDisabilities represents a candidate's disability information
type CandidateDisabilities struct {
	CandRegNumber    string    `db:"cand_reg_number" json:"cand_reg_number"`
	IsBlind         bool      `db:"is_blind" json:"is_blind"`
	IsDeaf          bool      `db:"is_deaf" json:"is_deaf"`
	OtherChallenges string    `db:"other_challenges" json:"other_challenges"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

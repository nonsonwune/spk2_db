package models

import "time"

// CandidateExamInfo represents a candidate's exam-related information
type CandidateExamInfo struct {
	CandRegNumber    string    `db:"cand_reg_number" json:"cand_reg_number"`
	ExamTown        string    `db:"exam_town" json:"exam_town"`
	ExamCentre      string    `db:"exam_centre" json:"exam_centre"`
	ExamNumber      string    `db:"exam_number" json:"exam_number"`
	MockStateID     int       `db:"mock_state_id" json:"mock_state_id"`
	MockTown        string    `db:"mock_town" json:"mock_town"`
	IsMockCandidate bool      `db:"is_mock_candidate" json:"is_mock_candidate"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	MockState       *State    `db:"-" json:"mock_state,omitempty"`
}

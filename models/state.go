package models

// State represents the state table
type State struct {
	ID           int     `db:"st_id" json:"id"`
	Abbreviation string  `db:"st_abreviation" json:"abbreviation"`
	Name         string  `db:"st_name" json:"name"`
	ELDS         bool    `db:"st_elds" json:"elds"`
}

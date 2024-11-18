package models

// Subject represents the subject table
type Subject struct {
	ID          int    `db:"su_id" json:"id"`
	Abbreviation string `db:"su_abrv" json:"abbreviation"`
	Name        string `db:"su_name" json:"name"`
}

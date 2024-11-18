package models

// LGA represents the lga table
type LGA struct {
	ID      int    `db:"lg_id" json:"id"`
	Name    string `db:"lg_name" json:"name"`
	StateID int    `db:"lg_st_id" json:"state_id"`
	State   *State `db:"-" json:"state,omitempty"`
}

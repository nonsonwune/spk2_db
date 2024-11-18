package models

// Faculty represents the faculty table
type Faculty struct {
	ID           int         `db:"fac_id" json:"id"`
	Name         string      `db:"fac_name" json:"name"`
	InstitutionID int        `db:"fac_inst_id" json:"institution_id"`
	Institution  *Institution `db:"-" json:"institution,omitempty"`
}

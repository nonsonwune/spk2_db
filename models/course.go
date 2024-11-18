package models

// Course represents the course table
type Course struct {
	ID        int      `db:"cou_id" json:"id"`
	Name      string   `db:"cou_name" json:"name"`
	FacultyID int      `db:"cou_fac_id" json:"faculty_id"`
	Faculty   *Faculty `db:"-" json:"faculty,omitempty"`
}

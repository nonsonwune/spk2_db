package models

import "database/sql"

// Candidate represents the candidates table
type Candidate struct {
	RegNumber     string         `db:"regnumber" json:"regnumber"`
	Year          int            `db:"year" json:"year"`
	MaritalStatus sql.NullString `db:"maritalstatus" json:"maritalstatus,omitempty"`
	Challenged    sql.NullString `db:"challenged" json:"challenged,omitempty"`
	Blind         sql.NullBool   `db:"blind" json:"blind,omitempty"`
	Deaf          sql.NullBool   `db:"deaf" json:"deaf,omitempty"`
	ExamTown      sql.NullString `db:"examtown" json:"examtown,omitempty"`
	ExamCentre    sql.NullString `db:"examcentre" json:"examcentre,omitempty"`
	ExamNo        sql.NullString `db:"examno" json:"examno,omitempty"`
	Address       sql.NullString `db:"address" json:"address,omitempty"`
	NoOfSittings  sql.NullInt64  `db:"noofsittings" json:"noofsittings,omitempty"`
	DateSaved     sql.NullString `db:"datesaved" json:"datesaved,omitempty"`
	TimeSaved     sql.NullString `db:"timesaved" json:"timesaved,omitempty"`
	MockCand      sql.NullBool   `db:"mockcand" json:"mockcand,omitempty"`
	MockState     sql.NullInt64  `db:"mockstate" json:"mockstate,omitempty"`
	MockTown      sql.NullString `db:"mocktown" json:"mocktown,omitempty"`
	DateCreated   sql.NullString `db:"datecreated" json:"datecreated,omitempty"`
	Email         sql.NullString `db:"email" json:"email,omitempty"`
	GSMNo         sql.NullString `db:"gsmno" json:"gsmno,omitempty"`
	Surname       sql.NullString `db:"surname" json:"surname,omitempty"`
	FirstName     sql.NullString `db:"firstname" json:"firstname,omitempty"`
	MiddleName    sql.NullString `db:"middlename" json:"middlename,omitempty"`
	DateOfBirth   sql.NullString `db:"dateofbirth" json:"dateofbirth,omitempty"`
	Gender        sql.NullString `db:"gender" json:"gender,omitempty"`
	StateCode     sql.NullInt64  `db:"statecode" json:"statecode,omitempty"`
	Subj1         sql.NullInt64  `db:"subj1" json:"subj1,omitempty"`
	Score1        sql.NullInt64  `db:"score1" json:"score1,omitempty"`
	Subj2         sql.NullInt64  `db:"subj2" json:"subj2,omitempty"`
	Score2        sql.NullInt64  `db:"score2" json:"score2,omitempty"`
	Subj3         sql.NullInt64  `db:"subj3" json:"subj3,omitempty"`
	Score3        sql.NullInt64  `db:"score3" json:"score3,omitempty"`
	Subj4         sql.NullInt64  `db:"subj4" json:"subj4,omitempty"`
	Score4        sql.NullInt64  `db:"score4" json:"score4,omitempty"`
	Aggregate     sql.NullInt64  `db:"aggregate" json:"aggregate,omitempty"`
}

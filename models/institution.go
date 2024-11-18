package models

import "time"

// Institution represents the institution table
type Institution struct {
    InID              string            `db:"inid" json:"inid"`
    InAbv             string            `db:"inabv" json:"inabv"`
    InName            string            `db:"inname" json:"inname"`
    InstStateID       int               `db:"inst_state_id" json:"inst_state_id"`
    AffiliatedStateID int               `db:"affiliated_state_id" json:"affiliated_state_id"`
    InTyp             int               `db:"intyp" json:"intyp"`
    InstCat           string            `db:"inst_cat" json:"inst_cat"`
    State             *State            `db:"-" json:"state,omitempty"`
    AffiliatedState   *State            `db:"-" json:"affiliated_state,omitempty"`
    Names             []InstitutionName `db:"-" json:"names,omitempty"`
}

// InstitutionName represents the institution_names table
type InstitutionName struct {
    InID          string     `db:"inid" json:"inid"`
    InAbv         string     `db:"inabv" json:"inabv"`
    InName        string     `db:"inname" json:"inname"`
    EffectiveFrom time.Time  `db:"effective_from" json:"effective_from"`
    EffectiveTo   *time.Time `db:"effective_to" json:"effective_to,omitempty"`
    ChangeReason  *string    `db:"change_reason" json:"change_reason,omitempty"`
}

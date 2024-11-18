package migrations

import (
	"database/sql"
	"fmt"
)

// InitSchema verifies that all required tables exist
func InitSchema(db *sql.DB) error {
	// We're using existing tables, so just verify they exist
	tables := []string{"state", "course", "institution", "lga", "subject"}
	
	for _, table := range tables {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)`
		
		err := db.QueryRow(query, table).Scan(&exists)
		if err != nil {
			return err
		}
		
		if !exists {
			return fmt.Errorf("required table %s does not exist", table)
		}
	}
	
	return nil
}

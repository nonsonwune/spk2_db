package importer

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"sync"
)

// Constants for configuration
const (
	DefaultBatchSize  = 1000
	DefaultWorkerCount = 4
	MaxRetries        = 3
)

// ColumnMapping defines how source columns map to destination columns
type ColumnMapping struct {
	SourceColumn      string
	DestinationColumn string
	TransformFunc     func(string) (interface{}, error)
}

// ImportConfig holds the configuration for data import
type ImportConfig struct {
	Year            int
	SourceFile      string
	IsAdmission     bool // New field to indicate if this is admission data
	RequiredColumns  []string
	BatchSize        int
	ValidateOnly     bool
	ColumnMappings   []ColumnMapping
	WorkerCount      int // Number of parallel workers to use
	InstitutionID    int
}

// StateMapper handles conversion between state names and IDs
type StateMapper struct {
	db        *sql.DB
	nameToID  map[string]int
	prepared  bool
	initOnce  sync.Once
}

func NewStateMapper(db *sql.DB) *StateMapper {
	return &StateMapper{
		db:       db,
		nameToID: make(map[string]int),
	}
}

func (sm *StateMapper) init() error {
	var err error
	sm.initOnce.Do(func() {
		// Initialize the map
		sm.nameToID = make(map[string]int)

		query := `SELECT st_id, st_name FROM state`  // Fixed: changed 'states' to 'state'
		rows, queryErr := sm.db.Query(query)
		if queryErr != nil {
			err = queryErr
			return
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var name string
			if scanErr := rows.Scan(&id, &name); scanErr != nil {
				err = scanErr
				return
			}
			
			// Store the name as is since it's already in uppercase
			sm.nameToID[name] = id
			
			// Add debug logging
			log.Printf("Loaded state mapping: %s -> %d", name, id)
		}
		sm.prepared = true
	})
	return err
}

func (sm *StateMapper) GetStateID(stateName string) (int, error) {
	if !sm.prepared {
		if err := sm.init(); err != nil {
			return 0, fmt.Errorf("failed to initialize state mapper: %v", err)
		}
	}

	// Convert input to uppercase to match database format
	cleanName := strings.ToUpper(strings.TrimSpace(stateName))
	
	// Handle special cases
	specialCases := map[string]string{
		"FCT ABUJA":                "FCT",
		"FEDERAL CAPITAL TERRITORY": "FCT",
		"ABUJA":                    "FCT",
		"AKWA-IBOM":               "AKWA IBOM",
		"CROSS-RIVER":             "CROSS RIVER",
		"NASARAWA":                "NASSARAWA",
		"AFRICA":                  "FOREIGNER",
		"WEST AFRICA":             "FOREIGNER",
		"REPUBLIC OF BENIN":       "COTONOU",
		"COTE D'IVORIE":           "COTE D VOIRE",
		"COTE D'IVOIRE":           "COTE D VOIRE",
	}

	if mapped, ok := specialCases[cleanName]; ok {
		cleanName = mapped
	}

	// Try direct lookup first
	if id, ok := sm.nameToID[cleanName]; ok {
		return id, nil
	}

	// If no exact match, try fuzzy matching
	rows, err := sm.db.Query("SELECT st_id, st_name FROM state")
	if err != nil {
		return 0, fmt.Errorf("error querying states: %v", err)
	}
	defer rows.Close()

	// Store all state mappings for logging
	stateMappings := make(map[string]int)
	var closestMatch string
	var closestID int
	var minDistance int = 1000

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		stateMappings[name] = id

		// Calculate Levenshtein distance
		distance := levenshteinDistance(cleanName, name)
		if distance < minDistance {
			minDistance = distance
			closestMatch = name
			closestID = id
		}
	}

	// If we found a reasonably close match (distance <= 2)
	if minDistance <= 2 {
		log.Printf("State %s matched to %s with ID: %d", stateName, closestMatch, closestID)
		return closestID, nil
	}

	// Log available mappings for debugging
	log.Printf("State not found: %s. Available mappings:", cleanName)
	for name, id := range stateMappings {
		log.Printf("  %s -> %d", name, id)
	}

	return 0, fmt.Errorf("state not found: %s", cleanName)
}

// CourseMapper handles validation of course codes and manages historical code tracking.
type CourseMapper struct {
	db          *sql.DB
	courseCodes map[string]bool
	prepared    bool
	initOnce    sync.Once
}

func NewCourseMapper(db *sql.DB) *CourseMapper {
	return &CourseMapper{
		db:          db,
		courseCodes: make(map[string]bool),
	}
}

func (cm *CourseMapper) init() error {
	var err error
	cm.initOnce.Do(func() {
		// Initialize the map
		cm.courseCodes = make(map[string]bool)

		query := `SELECT course_code FROM course`  // Fixed: changed table name from 'courses' to 'course'
		rows, queryErr := cm.db.Query(query)
		if queryErr != nil {
			err = queryErr
			return
		}
		defer rows.Close()

		for rows.Next() {
			var code string
			if scanErr := rows.Scan(&code); scanErr != nil {
				err = scanErr
				return
			}
			cm.courseCodes[code] = true
			
			// Add debug logging
			log.Printf("Loaded course code: %s", code)
		}
		cm.prepared = true
	})
	return err
}

func (cm *CourseMapper) UpsertCourse(courseCode, courseName string) error {
	query := `
        INSERT INTO course (course_code, course_name)  -- Fixed: changed table name and column names
        VALUES ($1, $2)
        ON CONFLICT (course_code) 
        DO UPDATE SET 
            course_name = COALESCE(EXCLUDED.course_name, course.course_name)
            WHERE course.course_name LIKE 'Course%'  -- Only update if current name is a code-based name
        RETURNING course_name;`

	var updatedName string
	err := cm.db.QueryRow(query, courseCode, courseName).Scan(&updatedName)
	if err != nil {
		return fmt.Errorf("failed to upsert course: %w", err)
	}

	// Log the change if it was updated
	if updatedName != "Course "+courseCode {
		log.Printf("Updated course %s: %s -> %s", courseCode, "Course "+courseCode, updatedName)
	}

	return nil
}

func (cm *CourseMapper) ValidateCourseCode(courseCode string, year int, institutionID int) error {
	if !cm.prepared {
		if err := cm.init(); err != nil {
			return fmt.Errorf("failed to initialize course mapper: %v", err)
		}
	}

	// Try exact match first
	if cm.courseCodes[courseCode] {
		return nil
	}

	// Store historical code
	err := cm.storeHistoricalCode(courseCode, year, institutionID)
	if err != nil {
		log.Printf("Warning: Failed to store historical code: %v", err)
	}

	// Return special error type for historical codes
	return &HistoricalCourseError{
		CourseCode:    courseCode,
		Year:          year,
		InstitutionID: institutionID,
	}
}

func (cm *CourseMapper) storeHistoricalCode(courseCode string, year int, institutionID int) error {
	query := `
        INSERT INTO historical_course_codes (year, old_course_code, institution_id, import_timestamp)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (year, old_course_code, institution_id) DO NOTHING
    `
	_, err := cm.db.Exec(query, year, courseCode, institutionID)
	return err
}

// HistoricalCourseError represents an error for historical course codes
type HistoricalCourseError struct {
	CourseCode    string
	Year          int
	InstitutionID int
}

func (e *HistoricalCourseError) Error() string {
	return fmt.Sprintf("historical course code: %s (Year: %d, Institution: %d)", 
		e.CourseCode, e.Year, e.InstitutionID)
}

// InstitutionMapper handles validation and transformation of institution codes
type InstitutionMapper struct {
	db           *sql.DB
	institutions map[string]string  // maps input codes to valid institution IDs
	prepared     bool
	initOnce     sync.Once
}

func NewInstitutionMapper(db *sql.DB) *InstitutionMapper {
	return &InstitutionMapper{
		db:           db,
		institutions: make(map[string]string),
	}
}

func (im *InstitutionMapper) init() error {
	var err error
	im.initOnce.Do(func() {
		// Initialize the map
		im.institutions = make(map[string]string)

		query := `SELECT inid, inabv, inname FROM institution`
		rows, queryErr := im.db.Query(query)
		if queryErr != nil {
			err = queryErr
			return
		}
		defer rows.Close()

		for rows.Next() {
			var id, abbrev, name string
			if scanErr := rows.Scan(&id, &abbrev, &name); scanErr != nil {
				err = scanErr
				return
			}
			
			// Store mappings
			im.institutions[id] = id // Direct mapping
			if abbrev != "" {
				im.institutions[abbrev] = id // Map abbreviation to ID
			}
			
			// Add debug logging
			log.Printf("Loaded institution mapping: %s -> %s (abbrev: %s)", id, id, abbrev)
		}
		im.prepared = true
	})
	return err
}

func (im *InstitutionMapper) GetInstitutionID(code string) (string, error) {
	if !im.prepared {
		if err := im.init(); err != nil {
			return "", fmt.Errorf("failed to initialize institution mapper: %v", err)
		}
	}

	// Clean and standardize input
	code = strings.TrimSpace(code)
	
	// Direct lookup
	if id, exists := im.institutions[code]; exists {
		return id, nil
	}

	// Log unmatched institution code
	log.Printf("Warning: No matching institution found for code: %s", code)
	return "", fmt.Errorf("invalid institution code: %s", code)
}

// DataImporter handles the import process
type DataImporter struct {
	db               *sql.DB
	config           ImportConfig
	stateMapper      *StateMapper
	courseMapper     *CourseMapper
	institutionMapper *InstitutionMapper
	failedIndices    map[int]error  // Track failed record indices
	mu               sync.Mutex     // Protect concurrent access to failedIndices
	columnMapping    map[string]string
}

func NewDataImporter(db *sql.DB, config ImportConfig) *DataImporter {
	if config.BatchSize == 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = DefaultWorkerCount
	}
	if config.ColumnMappings == nil {
		config.ColumnMappings = DefaultColumnMappings()
	}

	return &DataImporter{
		db:               db,
		config:           config,
		stateMapper:      NewStateMapper(db),
		courseMapper:     NewCourseMapper(db),
		institutionMapper: NewInstitutionMapper(db),
		failedIndices:    make(map[int]error),
	}
}

func DefaultColumnMappings() []ColumnMapping {
	return []ColumnMapping{
		{SourceColumn: "REGNUMBER", DestinationColumn: "regnumber"},
		{SourceColumn: "SURNAME", DestinationColumn: "surname"},
		{SourceColumn: "FIRSTNAME", DestinationColumn: "firstname"},
		{SourceColumn: "MIDDLENAME", DestinationColumn: "middlename"},
		{SourceColumn: "GENDER", DestinationColumn: "gender"},
		{SourceColumn: "EMAIL", DestinationColumn: "email"},
		{SourceColumn: "GSMNO", DestinationColumn: "gsmno"},
		{SourceColumn: "STATECODE", DestinationColumn: "statecode"},
		{SourceColumn: "LG_ID", DestinationColumn: "lg_id"},
		{SourceColumn: "INID", DestinationColumn: "inid"},
		{SourceColumn: "AGGREGATE", DestinationColumn: "aggregate"},
		{SourceColumn: "APP_COURSE1", DestinationColumn: "app_course1"},
		{SourceColumn: "IS_ADMITTED", DestinationColumn: "is_admitted"},
		{SourceColumn: "IS_DIRECT_ENTRY", DestinationColumn: "is_direct_entry"},
		{SourceColumn: "IS_BLIND", DestinationColumn: "is_blind"},
		{SourceColumn: "IS_DEAF", DestinationColumn: "is_deaf"},
		{SourceColumn: "IS_MOCK_CANDIDATE", DestinationColumn: "is_mock_candidate"},
		{SourceColumn: "MARITALSTATUS", DestinationColumn: "maritalstatus"},
		{SourceColumn: "ADDRESS", DestinationColumn: "address"},
		{SourceColumn: "NOOFSITTINGS", DestinationColumn: "noofsittings"},
		{SourceColumn: "MALPRACTICE", DestinationColumn: "malpractice"},
	}
}

func (di *DataImporter) initStateMapper() error {
	return di.stateMapper.init()
}

func (di *DataImporter) initCourseMapper() error {
	return di.courseMapper.init()
}

func (di *DataImporter) initInstitutionMapper() error {
	return di.institutionMapper.init()
}

// ColumnMatch represents a potential column match with confidence score
type ColumnMatch struct {
	SourceColumn      string
	DestinationColumn string
	Confidence       float64
}

// findBestColumnMatch uses fuzzy matching to find the best column match
func (di *DataImporter) findBestColumnMatch(sourceColumn string, requiredColumns []string) []ColumnMatch {
	matches := make([]ColumnMatch, 0)
	
	// Normalize source column
	normalizedSource := strings.ToLower(strings.TrimSpace(sourceColumn))
	normalizedSource = strings.ReplaceAll(normalizedSource, "_", "")
	normalizedSource = strings.ReplaceAll(normalizedSource, " ", "")
	
	for _, destColumn := range requiredColumns {
		// Normalize destination column
		normalizedDest := strings.ToLower(strings.TrimSpace(destColumn))
		normalizedDest = strings.ReplaceAll(normalizedDest, "_", "")
		normalizedDest = strings.ReplaceAll(normalizedDest, " ", "")
		
		// Calculate similarity score
		distance := levenshteinDistance(normalizedSource, normalizedDest)
		maxLen := float64(max(len(normalizedSource), len(normalizedDest)))
		confidence := 1.0 - float64(distance)/maxLen
		
		if confidence > 0.6 { // Only consider matches with >60% confidence
			matches = append(matches, ColumnMatch{
				SourceColumn:      sourceColumn,
				DestinationColumn: destColumn,
				Confidence:       confidence,
			})
		}
	}
	
	// Sort matches by confidence
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})
	
	return matches
}

// validateHeaders checks if all required columns are present with user interaction
func (di *DataImporter) validateHeaders(headers []string) error {
	missingColumns := make([]string, 0)
	di.columnMapping = make(map[string]string)
	
	for _, required := range di.config.RequiredColumns {
		found := false
		exactMatch := getColumnIndex(headers, required) != -1
		
		if exactMatch {
			di.columnMapping[required] = required
			found = true
			continue
		}
		
		// Try fuzzy matching
		matches := di.findBestColumnMatch(required, headers)
		if len(matches) > 0 {
			// Ask user for confirmation if multiple matches found
			if len(matches) > 1 {
				fmt.Printf("\nMultiple potential matches found for column '%s':\n", required)
				for i, match := range matches {
					fmt.Printf("%d. %s (confidence: %.2f%%)\n", i+1, match.SourceColumn, match.Confidence*100)
				}
				fmt.Print("Enter number to select match (0 to skip): ")
				var choice int
				fmt.Scanln(&choice)
				
				if choice > 0 && choice <= len(matches) {
					di.columnMapping[required] = matches[choice-1].SourceColumn
					found = true
				}
			} else if matches[0].Confidence > 0.8 { // Auto-accept high confidence matches
				di.columnMapping[required] = matches[0].SourceColumn
				found = true
				fmt.Printf("Automatically mapped '%s' to '%s' (%.2f%% confidence)\n", 
					required, matches[0].SourceColumn, matches[0].Confidence*100)
			} else {
				// Ask for confirmation for lower confidence matches
				fmt.Printf("\nPotential match found for column '%s':\n", required)
				fmt.Printf("'%s' (confidence: %.2f%%)\n", matches[0].SourceColumn, matches[0].Confidence*100)
				fmt.Print("Accept this match? (y/n): ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) == "y" {
					di.columnMapping[required] = matches[0].SourceColumn
					found = true
				}
			}
		}
		
		if !found {
			missingColumns = append(missingColumns, required)
		}
	}
	
	if len(missingColumns) > 0 {
		return fmt.Errorf("missing required columns: %v", missingColumns)
	}
	
	return nil
}

type ImportResult struct {
    ChunkIndex   int
    SuccessCount int
    FailedCount  int
    Errors       []error
}

// ImportData is a package-level function that creates a new importer and imports data
func ImportData(ctx context.Context, db *sql.DB, config ImportConfig, reader *csv.Reader) error {
    importer := NewDataImporter(db, config)
    if importer.config.ColumnMappings == nil {
        importer.config.ColumnMappings = DefaultColumnMappings()
    }
    return importer.ImportData(ctx, reader)
}

// ImportCourses is a package-level function that creates a new importer and imports course data
func ImportCourses(ctx context.Context, db *sql.DB, config ImportConfig, reader *csv.Reader) error {
    importer := NewDataImporter(db, config)
    return importer.ImportCourses(ctx, reader)
}

func (di *DataImporter) ImportData(ctx context.Context, reader *csv.Reader) error {
    // Read headers
    headers, err := reader.Read()
    if err != nil {
        return fmt.Errorf("error reading headers: %v", err)
    }

    // Initialize mappers
    if err := di.initStateMapper(); err != nil {
        return fmt.Errorf("error initializing state mapper: %v", err)
    }
    if err := di.initCourseMapper(); err != nil {
        return fmt.Errorf("error initializing course mapper: %v", err)
    }
    if err := di.initInstitutionMapper(); err != nil {
        return fmt.Errorf("error initializing institution mapper: %v", err)
    }

    // Prepare column mappings
    if err := di.validateHeaders(headers); err != nil {
        return fmt.Errorf("invalid headers: %v", err)
    }

    // Start a transaction
    tx, err := di.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
    if err != nil {
        return fmt.Errorf("error starting transaction: %v", err)
    }
    defer tx.Rollback() // Will be ignored if transaction is committed

    // Prepare the insert statement
    stmt, err := di.prepareInsertStatement(tx)
    if err != nil {
        return fmt.Errorf("error preparing statement: %v", err)
    }
    defer stmt.Close()

    // Process records in batches
    batchSize := 1000 // Adjust based on your needs
    batch := make([][]string, 0, batchSize)
    totalProcessed := 0
    successCount := 0
    failedCount := 0
    var lastError error

    for {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return fmt.Errorf("import cancelled: %v", ctx.Err())
        default:
        }

        // Read record
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Printf("Error reading record: %v", err)
            failedCount++
            continue
        }

        batch = append(batch, record)
        
        // Process batch when it's full or on last record
        if len(batch) >= batchSize {
            result := di.processBatch(ctx, batch, headers, totalProcessed, stmt)
            successCount += result.SuccessCount
            failedCount += result.FailedCount
            if len(result.Errors) > 0 {
                lastError = result.Errors[len(result.Errors)-1]
            }
            
            // Log progress
            totalProcessed += len(batch)
            if totalProcessed%10000 == 0 {
                log.Printf("Processed %d records. Success: %d, Failed: %d", 
                    totalProcessed, successCount, failedCount)
            }
            
            // Commit batch transaction
            if err := tx.Commit(); err != nil {
                return fmt.Errorf("error committing batch: %v", err)
            }
            
            // Start new transaction for next batch
            tx, err = di.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
            if err != nil {
                return fmt.Errorf("error starting new batch transaction: %v", err)
            }
            
            // Prepare new statement for next batch
            stmt, err = di.prepareInsertStatement(tx)
            if err != nil {
                return fmt.Errorf("error preparing statement for new batch: %v", err)
            }
            
            batch = batch[:0] // Clear batch
        }
    }

    // Process remaining records
    if len(batch) > 0 {
        result := di.processBatch(ctx, batch, headers, totalProcessed, stmt)
        successCount += result.SuccessCount
        failedCount += result.FailedCount
        if len(result.Errors) > 0 {
            lastError = result.Errors[len(result.Errors)-1]
        }
        totalProcessed += len(batch)
        
        // Commit final batch
        if err := tx.Commit(); err != nil {
            return fmt.Errorf("error committing final batch: %v", err)
        }
    }

    // Print summary
    di.printImportSummary(successCount, failedCount, []error{lastError})

    if failedCount > 0 {
        return fmt.Errorf("import completed with %d failures, last error: %v", 
            failedCount, lastError)
    }

    return nil
}

func (di *DataImporter) processBatch(ctx context.Context, records [][]string, headers []string, startIndex int, stmt *sql.Stmt) ImportResult {
    result := ImportResult{
        ChunkIndex: startIndex,
    }

    for _, record := range records {
        // Check context cancellation
        select {
        case <-ctx.Done():
            result.Errors = append(result.Errors, ctx.Err())
            return result
        default:
        }

        // Transform and insert record
        values, err := di.transformRecord(headers, record)
        if err != nil {
            result.FailedCount++
            result.Errors = append(result.Errors, err)
            log.Printf("Error transforming record at index %d: %v", startIndex+result.FailedCount+result.SuccessCount, err)
            continue
        }

        // Execute insert
        if _, err := stmt.Exec(values...); err != nil {
            result.FailedCount++
            result.Errors = append(result.Errors, err)
            log.Printf("Error inserting record at index %d: %v", startIndex+result.FailedCount+result.SuccessCount, err)
        } else {
            result.SuccessCount++
        }
    }

    return result
}

func (di *DataImporter) prepareInsertStatement(tx *sql.Tx) (*sql.Stmt, error) {
    // Build column list
    columns := make([]string, 0, len(di.config.ColumnMappings))
    placeholders := make([]string, 0, len(di.config.ColumnMappings))
    for i, mapping := range di.config.ColumnMappings {
        columns = append(columns, mapping.DestinationColumn)
        placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
    }

    // Build COALESCE-based update clause for each column
    updateClauses := make([]string, 0, len(columns))
    for _, col := range columns {
        if col != "regnumber" { // Skip primary key in updates
            // Use COALESCE to keep existing non-null values if new value is null
            updateClauses = append(updateClauses, 
                fmt.Sprintf("%s = COALESCE(NULLIF(EXCLUDED.%s, ''), %s.%s)", 
                    col, col, "candidate", col))
        }
    }

    // Prepare the statement with COALESCE-based updates
    query := fmt.Sprintf(
        `INSERT INTO candidate (%s) 
         VALUES (%s) 
         ON CONFLICT (regnumber) 
         DO UPDATE SET %s`,
        strings.Join(columns, ", "),
        strings.Join(placeholders, ", "),
        strings.Join(updateClauses, ", "),
    )

    stmt, err := tx.Prepare(query)
    if err != nil {
        return nil, fmt.Errorf("error preparing statement: %v", err)
    }

    return stmt, nil
}

func levenshteinDistance(s1, s2 string) int {
    if len(s1) == 0 {
        return len(s2)
    }
    if len(s2) == 0 {
        return len(s1)
    }

    matrix := make([][]int, len(s1)+1)
    for i := range matrix {
        matrix[i] = make([]int, len(s2)+1)
    }

    for i := 0; i <= len(s1); i++ {
        matrix[i][0] = i
    }
    for j := 0; j <= len(s2); j++ {
        matrix[0][j] = j
    }

    for i := 1; i <= len(s1); i++ {
        for j := 1; j <= len(s2); j++ {
            if s1[i-1] == s2[j-1] {
                matrix[i][j] = matrix[i-1][j-1]
            } else {
                matrix[i][j] = min(
                    matrix[i-1][j]+1,
                    matrix[i][j-1]+1,
                    matrix[i-1][j-1]+1,
                )
            }
        }
    }

    return matrix[len(s1)][len(s2)]
}

func min(numbers ...int) int {
    if len(numbers) == 0 {
        return 0
    }
    result := numbers[0]
    for _, num := range numbers[1:] {
        if num < result {
            result = num
        }
    }
    return result
}

func getColumnIndex(headers []string, columnName string) int {
    for i, header := range headers {
        normalizedHeader := strings.ToLower(strings.TrimSpace(header))
        normalizedColumn := strings.ToLower(strings.TrimSpace(columnName))
        
        if normalizedHeader == normalizedColumn {
            return i
        }
        
        headerNoSpace := strings.ReplaceAll(normalizedHeader, " ", "")
        columnNoSpace := strings.ReplaceAll(normalizedColumn, " ", "")
        if headerNoSpace == columnNoSpace {
            return i
        }
    }
    return -1
}

func (di *DataImporter) transformRecord(headers []string, record []string) ([]interface{}, error) {
    values := make([]interface{}, len(di.config.ColumnMappings))
    
    for i, mapping := range di.config.ColumnMappings {
        idx := getColumnIndex(headers, mapping.SourceColumn)
        if idx == -1 || idx >= len(record) {
            values[i] = nil
            continue
        }
        
        value := strings.TrimSpace(record[idx])
        if value == "" {
            values[i] = nil
            continue
        }
        
        switch mapping.DestinationColumn {
        case "regnumber", "surname", "firstname", "middlename", "email", "gsmno":
            values[i] = value
        case "gender":
            if strings.EqualFold(value, "M") || strings.EqualFold(value, "MALE") {
                values[i] = "M"
            } else if strings.EqualFold(value, "F") || strings.EqualFold(value, "FEMALE") {
                values[i] = "F"
            } else {
                values[i] = nil
            }
        case "is_admitted", "is_direct_entry", "is_blind", "is_deaf", "is_mock_candidate":
            if strings.EqualFold(value, "yes") || strings.EqualFold(value, "true") || value == "1" {
                values[i] = true
            } else {
                values[i] = false
            }
        default:
            values[i] = value
        }
    }
    
    return values, nil
}

func (di *DataImporter) printImportSummary(successCount, failedCount int, errors []error) {
    log.Printf("\nImport Summary:")
    log.Printf("Total Records Processed: %d", successCount+failedCount)
    log.Printf("Successfully Imported: %d (%.2f%%)", 
        successCount, 
        float64(successCount)/float64(successCount+failedCount)*100)
    log.Printf("Failed Records: %d (%.2f%%)", 
        failedCount,
        float64(failedCount)/float64(successCount+failedCount)*100)

    if len(errors) > 0 {
        log.Printf("\nLast Error: %v", errors[0])
    }
}

func (di *DataImporter) ImportCourses(ctx context.Context, reader *csv.Reader) error {
    // Skip header row
    header, err := reader.Read()
    if err != nil {
        return fmt.Errorf("failed to read header: %v", err)
    }

    // Initialize column indices
    columnIndices := make(map[string]int)
    for i, col := range header {
        columnIndices[strings.ToUpper(strings.TrimSpace(col))] = i
    }

    // Process records in batches
    batch := make([][]string, 0, di.config.BatchSize)
    rowNum := 1 // Start after header

    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("error reading record at row %d: %v", rowNum, err)
        }

        batch = append(batch, record)
        
        if len(batch) >= di.config.BatchSize {
            if err := di.processCoursesBatch(ctx, batch, columnIndices); err != nil {
                return fmt.Errorf("error processing batch at row %d: %v", rowNum, err)
            }
            batch = batch[:0]
        }
        
        rowNum++
    }

    // Process remaining records
    if len(batch) > 0 {
        if err := di.processCoursesBatch(ctx, batch, columnIndices); err != nil {
            return fmt.Errorf("error processing final batch: %v", err)
        }
    }

    return nil
}

func (di *DataImporter) processCoursesBatch(ctx context.Context, batch [][]string, columnIndices map[string]int) error {
    tx, err := di.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback()

    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO course (
            code,
            name,
            description,
            faculty_id,
            created_at,
            updated_at
        ) VALUES ($1, $2, $3, $4, NOW(), NOW())
        ON CONFLICT (code) DO UPDATE SET
            name = EXCLUDED.name,
            description = COALESCE(EXCLUDED.description, course.description),
            faculty_id = COALESCE(EXCLUDED.faculty_id, course.faculty_id),
            updated_at = NOW()
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare statement: %v", err)
    }
    defer stmt.Close()

    for _, record := range batch {
        code := strings.TrimSpace(record[columnIndices["CODE"]])
        name := strings.TrimSpace(record[columnIndices["NAME"]])
        description := strings.TrimSpace(record[columnIndices["DESCRIPTION"]])
        facultyID := strings.TrimSpace(record[columnIndices["FACULTY_ID"]])

        if code == "" || name == "" {
            continue // Skip invalid records
        }

        _, err = stmt.ExecContext(ctx, code, name, description, facultyID)
        if err != nil {
            return fmt.Errorf("failed to insert course %s: %v", code, err)
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %v", err)
    }

    return nil
}

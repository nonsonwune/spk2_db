package importer

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
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

		query := `SELECT st_id, st_name FROM states`
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
	rows, err := sm.db.Query("SELECT st_id, st_name FROM states")
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

		query := `SELECT corcode FROM courses`
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
        INSERT INTO courses (corcode, "COURSE NAME")
        VALUES ($1, $2)
        ON CONFLICT (corcode) 
        DO UPDATE SET 
            "COURSE NAME" = EXCLUDED."COURSE NAME"
            WHERE courses."COURSE NAME" LIKE 'Course%'  -- Only update if current name is a code-based name
        RETURNING "COURSE NAME";`

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

		query := `SELECT inid, inabv, inname FROM institutions`
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
	return &DataImporter{
		db:           db,
		config:       config,
		stateMapper:  NewStateMapper(db),
		courseMapper: NewCourseMapper(db),
		institutionMapper: NewInstitutionMapper(db),
		failedIndices:    make(map[int]error),
	}
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

// ImportResult represents the result of importing a chunk of records
type ImportResult struct {
    SuccessCount int
    FailedCount  int
    ChunkIndex   int
    Errors       []error
}

func ImportData(ctx context.Context, db *sql.DB, config ImportConfig, reader *csv.Reader) error {
    importer := NewDataImporter(db, config)
    
    // If no column mappings are provided, use the defaults
    if len(config.ColumnMappings) == 0 {
        importer.config.ColumnMappings = importer.DefaultColumnMappings()
    }

    return importer.ImportData(ctx, reader)
}

func (d *DataImporter) ImportData(ctx context.Context, reader *csv.Reader) error {
    // Check if context is cancelled
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Initialize mappers
    if err := d.initStateMapper(); err != nil {
        return fmt.Errorf("error initializing state mapper: %w", err)
    }
    if err := d.initCourseMapper(); err != nil {
        return fmt.Errorf("error initializing course mapper: %w", err)
    }
    if err := d.initInstitutionMapper(); err != nil {
        return fmt.Errorf("error initializing institution mapper: %w", err)
    }

    // Read headers
    headers, err := reader.Read()
    if err != nil {
        return fmt.Errorf("error reading headers: %w", err)
    }

    // Validate headers
    if err := d.validateHeaders(headers); err != nil {
        return fmt.Errorf("header validation failed: %w", err)
    }

    // Start transaction
    tx, err := d.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("error starting transaction: %w", err)
    }
    defer tx.Rollback()

    // Prepare insert statement
    stmt, err := d.prepareInsertStatement(tx)
    if err != nil {
        return fmt.Errorf("error preparing statement: %w", err)
    }
    defer stmt.Close()

    // Process records in batches
    var records [][]string
    batchSize := d.config.BatchSize
    if batchSize == 0 {
        batchSize = DefaultBatchSize
    }

    records = make([][]string, 0, batchSize)
    recordIndex := 0
    var totalSuccess, totalFailed int
    var allErrors []error

    for {
        // Check context before reading next record
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Printf("Error reading record: %v", err)
            continue
        }

        records = append(records, record)
        recordIndex++

        if len(records) >= batchSize {
            result := d.processBatch(ctx, records, headers, recordIndex-len(records), stmt)
            totalSuccess += result.SuccessCount
            totalFailed += result.FailedCount
            allErrors = append(allErrors, result.Errors...)
            records = records[:0]
        }
    }

    // Process remaining records
    if len(records) > 0 {
        result := d.processBatch(ctx, records, headers, recordIndex-len(records), stmt)
        totalSuccess += result.SuccessCount
        totalFailed += result.FailedCount
        allErrors = append(allErrors, result.Errors...)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("error committing transaction: %w", err)
    }

    // Print summary
    d.printImportSummary(totalSuccess, totalFailed, allErrors)

    if totalFailed > 0 {
        return fmt.Errorf("import completed with %d failures", totalFailed)
    }

    return nil
}

func (d *DataImporter) processBatch(ctx context.Context, records [][]string, headers []string, startIndex int, stmt *sql.Stmt) ImportResult {
    result := ImportResult{
        ChunkIndex: startIndex,
    }

    for i, record := range records {
        select {
        case <-ctx.Done():
            return result
        default:
            if err := d.processRecord(ctx, record, headers, stmt); err != nil {
                result.FailedCount++
                result.Errors = append(result.Errors, fmt.Errorf("row %d: %v", startIndex+i+1, err))
                d.failedIndices[startIndex+i] = err
            } else {
                result.SuccessCount++
            }
        }
    }

    return result
}

func (d *DataImporter) processRecord(ctx context.Context, record []string, headers []string, stmt *sql.Stmt) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        values, err := d.transformRecord(headers, record)
        if err != nil {
            return err
        }

        for i := 0; i < MaxRetries; i++ {
            if err := d.executeInsert(stmt, values); err != nil {
                if i == MaxRetries-1 {
                    return err
                }
                time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
                continue
            }
            break
        }

        return nil
    }
}

func (d *DataImporter) executeInsert(stmt *sql.Stmt, values []interface{}) error {
    _, err := stmt.Exec(values...)
    if err != nil {
        return &ImportError{
            Code:    "INSERT_FAILED",
            Message: err.Error(),
            Context: map[string]string{
                "values": fmt.Sprintf("%v", values),
            },
        }
    }
    return nil
}

func (d *DataImporter) initStateMapper() error {
    d.stateMapper = NewStateMapper(d.db)
    return d.stateMapper.init()
}

func (d *DataImporter) initCourseMapper() error {
    d.courseMapper = NewCourseMapper(d.db)
    return d.courseMapper.init()
}

func (d *DataImporter) initInstitutionMapper() error {
    d.institutionMapper = NewInstitutionMapper(d.db)
    return d.institutionMapper.init()
}

type ChunkResult struct {
    FailedImports []FailedImport
    ProcessedRows int
    ChunkIndex    int
}

func (d *DataImporter) processChunk(records [][]string, headers []string, startIndex int) ChunkResult {
    result := ChunkResult{
        ChunkIndex: startIndex,
    }

    // Get column indexes based on mappings
    regNumberIdx := getColumnIndex(headers, "RegNumber")
    courseCodeIdx := getColumnIndex(headers, "CourseCode")
    stateIdx := getColumnIndex(headers, "State")
    genderIdx := getColumnIndex(headers, "Gender")

    for _, record := range records {
        result.ProcessedRows++
        
        // Try to transform the record
        _, err := d.transformRecord(headers, record)
        if err != nil {
            failed := FailedImport{
                RowData:    record,
                FailReason: err.Error(),
            }

            // Get specific fields if they exist
            if regNumberIdx >= 0 && regNumberIdx < len(record) {
                failed.RegNumber = record[regNumberIdx]
            }
            if courseCodeIdx >= 0 && courseCodeIdx < len(record) {
                failed.CourseCode = record[courseCodeIdx]
            }
            if stateIdx >= 0 && stateIdx < len(record) {
                failed.StateCode = record[stateIdx]
            }
            if genderIdx >= 0 && genderIdx < len(record) {
                failed.Gender = record[genderIdx]
            }

            result.FailedImports = append(result.FailedImports, failed)
        }
    }

    return result
}

func (d *DataImporter) AnalyzeFailedImports(filename string) ([]FailedImport, error) {
    // Initialize the column mappings if not already done
    if d.config.ColumnMappings == nil {
        d.config.ColumnMappings = d.DefaultColumnMappings()
    }

    // Initialize the state mapper if not already done
    if d.stateMapper == nil {
        if err := d.initStateMapper(); err != nil {
            return nil, fmt.Errorf("failed to initialize state mapper: %v", err)
        }
    }

    // Initialize the course mapper if not already done
    if d.courseMapper == nil {
        if err := d.initCourseMapper(); err != nil {
            return nil, fmt.Errorf("failed to initialize course mapper: %v", err)
        }
    }

    if d.institutionMapper == nil {
        if err := d.initInstitutionMapper(); err != nil {
            return nil, fmt.Errorf("failed to initialize institution mapper: %v", err)
        }
    }

    file, err := os.Open(filename)
    if err != nil {
        return nil, fmt.Errorf("error opening file: %v", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
    headers, err := reader.Read()
    if err != nil {
        return nil, fmt.Errorf("error reading headers: %v", err)
    }

    // Read all records
    allRecords, err := reader.ReadAll()
    if err != nil {
        return nil, fmt.Errorf("error reading records: %v", err)
    }

    // Use configured worker count or default to 4
    workerCount := d.config.WorkerCount
    if workerCount <= 0 {
        workerCount = DefaultWorkerCount
    }
    
    recordsPerWorker := (len(allRecords) + workerCount - 1) / workerCount
    
    // Create channels for results and errors
    results := make(chan ChunkResult, workerCount)
    var wg sync.WaitGroup

    // Process chunks in parallel
    for i := 0; i < workerCount; i++ {
        start := i * recordsPerWorker
        end := start + recordsPerWorker
        if end > len(allRecords) {
            end = len(allRecords)
        }

        if start >= len(allRecords) {
            break
        }

        chunk := allRecords[start:end]
        wg.Add(1)
        
        go func(chunk [][]string, startIndex int) {
            defer wg.Done()
            result := d.processChunk(chunk, headers, startIndex)
            results <- result
        }(chunk, start)
    }

    // Start a goroutine to close results channel after all workers are done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    var allFailedImports []FailedImport
    totalProcessed := 0

    // Collect all results from the channel
    for result := range results {
        allFailedImports = append(allFailedImports, result.FailedImports...)
        totalProcessed += result.ProcessedRows
    }

    // Print analysis
    if len(allFailedImports) > 0 {
        log.Printf("\nAnalysis of Failed Imports:")
        log.Printf("Total Records Processed: %d", totalProcessed)
        log.Printf("Total Failed Records: %d (%.2f%%)\n", 
            len(allFailedImports), 
            float64(len(allFailedImports))/float64(totalProcessed)*100)

        // Count failures by reason
        reasonCounts := make(map[string]int)
        for _, f := range allFailedImports {
            reasonCounts[f.FailReason]++
        }

        log.Printf("\nFailure Reasons:")
        for reason, count := range reasonCounts {
            log.Printf("- %s: %d occurrences (%.2f%%)", 
                reason, 
                count,
                float64(count)/float64(len(allFailedImports))*100)
        }

        // Show sample of failed records
        log.Printf("\nSample Failed Records (up to 10):")
        for i := 0; i < min(10, len(allFailedImports)); i++ {
            f := allFailedImports[i]
            log.Printf("Row %d:", i+1)
            log.Printf("  RegNumber: %s", f.RegNumber)
            log.Printf("  CourseCode: %s", f.CourseCode)
            log.Printf("  State: %s", f.StateCode)
            log.Printf("  Gender: %s", f.Gender)
            log.Printf("  Reason: %s\n", f.FailReason)
        }
    } else {
        log.Printf("\nNo failed imports found in the file.")
    }

    return allFailedImports, nil
}

type FailedImport struct {
    RegNumber   string
    CourseCode  string
    StateCode   string
    Gender      string
    FailReason  string
    ErrorCode   string    // Added field for error tracking
    Timestamp   time.Time // Added field for error timing
    RowNumber   int      // Added field for source tracking
    SourceFile  string   // Added field for file tracking
    RowData     []string
}

type ImportError struct {
    Code      string
    Message   string
    Timestamp time.Time
    Context   map[string]string
}

func (e *ImportError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (d *DataImporter) SaveFailedRecords(records [][]string, headers []string, failedIndices map[int]error) error {
    if len(failedIndices) == 0 {
        return nil
    }

    // Create failed records directory if it doesn't exist
    failedDir := "failed_imports"
    if err := os.MkdirAll(failedDir, 0755); err != nil {
        return fmt.Errorf("error creating failed_imports directory: %v", err)
    }

    // Create failed records file with timestamp
    timestamp := time.Now().Format("20060102_150405")
    failedFile := filepath.Join(failedDir, fmt.Sprintf("failed_records_%s.csv", timestamp))
    
    file, err := os.Create(failedFile)
    if err != nil {
        return fmt.Errorf("error creating failed records file: %v", err)
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Write headers
    if err := writer.Write(append(headers, "Error")); err != nil {
        return fmt.Errorf("error writing headers: %v", err)
    }

    // Write failed records with error messages
    for idx, err := range failedIndices {
        if idx < len(records) {
            record := append(records[idx], err.Error())
            if err := writer.Write(record); err != nil {
                return fmt.Errorf("error writing record: %v", err)
            }
        }
    }

    log.Printf("Failed records saved to: %s", failedFile)
    return nil
}

func (d *DataImporter) transformState(s string) (interface{}, error) {
    if s == "" {
        return nil, nil
    }

    // Clean the input
    s = strings.TrimSpace(strings.ToUpper(s))
    s = strings.ReplaceAll(s, "-", " ")

    // Map foreign states to their standard names
    foreignStates := map[string]string{
        "COTE D VOIRE": "COTE D'IVOIRE",
        "COTE D'VOIRE": "COTE D'IVOIRE",
        "COTE DE VOIRE": "COTE D'IVOIRE",
        "COTE DIVOIRE": "COTE D'IVOIRE",
        "IVORY COAST": "COTE D'IVOIRE",
        "CAMEROUN": "CAMEROON",
        "CAMEROONS": "CAMEROON",
        "LONDON": "UNITED KINGDOM",
        "UK": "UNITED KINGDOM",
        "BRITAIN": "UNITED KINGDOM",
        "ENGLAND": "UNITED KINGDOM",
        "JEDDAH": "SAUDI ARABIA",
        "SA": "SAUDI ARABIA",
        "RSA": "SOUTH AFRICA",
        "GHANA": "GHANA",
        "COTONOU": "BENIN REPUBLIC",
        "BENIN": "BENIN REPUBLIC",
    }

    // Map foreign state names if found
    if mappedState, ok := foreignStates[s]; ok {
        s = mappedState
    }

    // Try exact match first
    id, err := d.stateMapper.GetStateID(s)
    if err == nil {
        return id, nil
    }

    // If not found, try fuzzy matching
    // Split into words and try each word
    s = strings.ReplaceAll(s, "-", " ")
    words := strings.Fields(s)
    
    // Try each word
    for _, word := range words {
        id, err := d.stateMapper.GetStateID(word)
        if err == nil {
            return id, nil
        }
    }

    // Default to ID STATE if no match found
    return 99, nil
}

func (d *DataImporter) transformCourse(s string) (interface{}, error) {
    if s == "" {
        return nil, nil
    }

    // Clean and standardize the course code
    s = strings.TrimSpace(strings.ToUpper(s))
    
    // Remove any non-alphanumeric characters
    s = strings.Map(func(r rune) rune {
        if unicode.IsLetter(r) || unicode.IsNumber(r) {
            return r
        }
        return -1
    }, s)

    // Ensure the course code follows the correct format (e.g., 100202H)
    if len(s) != 7 {
        return nil, fmt.Errorf("invalid course code format: %s", s)
    }

    // Check if the course exists
    var exists bool
    err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM courses WHERE corcode = $1)", s).Scan(&exists)
    if err != nil {
        return nil, fmt.Errorf("error checking course existence: %v", err)
    }

    if !exists {
        // Course doesn't exist, create it
        _, err = d.db.Exec(`
            INSERT INTO courses (corcode, "COURSE NAME") 
            VALUES ($1, $2) 
            ON CONFLICT (corcode) DO NOTHING`, 
            s, fmt.Sprintf("Course %s", s))
        if err != nil {
            return nil, fmt.Errorf("error creating course: %v", err)
        }
    }

    return s, nil
}

func (d *DataImporter) transformLGA(s string) (interface{}, error) {
    if s == "" {
        return nil, nil
    }

    // Try to convert directly to integer first
    if lgID, err := strconv.Atoi(s); err == nil {
        // Verify the LGA ID exists
        var exists bool
        err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM lga WHERE lg_id = $1)", lgID).Scan(&exists)
        if err != nil {
            return nil, fmt.Errorf("error checking LGA existence: %v", err)
        }
        if exists {
            return lgID, nil
        }
    }

    // If not a valid ID, try to find by name
    s = strings.TrimSpace(strings.ToUpper(s))
    s = strings.ReplaceAll(s, "-", " ")

    var lgID int
    err := d.db.QueryRow(`
        SELECT lg_id 
        FROM lga 
        WHERE UPPER(lg_name) = $1 
        OR UPPER(lg_abreviation) = $1
        LIMIT 1`, s).Scan(&lgID)

    if err == nil {
        return lgID, nil
    }

    // If exact match fails, try fuzzy matching
    rows, err := d.db.Query(`
        SELECT lg_id, lg_name, lg_abreviation 
        FROM lga 
        WHERE lg_st_id = (
            SELECT lg_st_id 
            FROM lga 
            GROUP BY lg_st_id 
            ORDER BY COUNT(*) DESC 
            LIMIT 1
        )`)
    if err != nil {
        return nil, fmt.Errorf("error querying LGAs: %v", err)
    }
    defer rows.Close()

    var bestMatch struct {
        id       int
        distance int
    }
    bestMatch.distance = 1000

    for rows.Next() {
        var id int
        var name, abbr string
        if err := rows.Scan(&id, &name, &abbr); err != nil {
            continue
        }

        // Try matching against both name and abbreviation
        nameDist := levenshteinDistance(s, strings.ToUpper(name))
        abbrDist := levenshteinDistance(s, strings.ToUpper(abbr))

        // Use the better match
        dist := min(nameDist, abbrDist)
        if dist < bestMatch.distance {
            bestMatch.id = id
            bestMatch.distance = dist
        }
    }

    // If we found a reasonably close match
    if bestMatch.distance <= 3 {
        return bestMatch.id, nil
    }

    // Default to a common LGA if no match found
    return 101, nil // Default to first LGA (Aba North)
}

func (d *DataImporter) transformInstitution(s string) (interface{}, error) {
    if s == "" {
        return nil, nil
    }

    // Clean and standardize input
    s = strings.TrimSpace(s)
    
    // Direct lookup
    id, err := d.institutionMapper.GetInstitutionID(s)
    if err != nil {
        return nil, fmt.Errorf("error transforming institution code: %v", err)
    }

    return id, nil
}

func (d *DataImporter) transformAdmission(s string) (interface{}, error) {
    if d.config.IsAdmission {
        return true, nil
    }
    return nil, nil
}

func transformBool(s string) (interface{}, error) {
    if s == "" {
        return nil, nil
    }
    s = strings.ToLower(strings.TrimSpace(s))
    switch s {
    case "true", "t", "yes", "y", "1":
        return true, nil
    case "false", "f", "no", "n", "0":
        return false, nil
    default:
        return false, nil // Default to false for any other value
    }
}

func transformString(s string) (interface{}, error) {
    return strings.TrimSpace(s), nil
}

func transformInt(s string) (interface{}, error) {
	if s == "" {
		return nil, nil
	}
	return strconv.Atoi(s)
}

func transformDate(s string) (interface{}, error) {
	if s == "" {
		return nil, nil
	}
	// Add date format validation if needed
	return s, nil
}

type ImportStats struct {
    TotalProcessed  int
    ValidRecords    int
    SkippedRecords  int
    ErrorsByType    map[string]int
    InvalidCourses  map[string]int
}

func NewImportStats() *ImportStats {
    return &ImportStats{
        ErrorsByType:   make(map[string]int),
        InvalidCourses: make(map[string]int),
    }
}

func (s *ImportStats) AddError(errType string) {
    s.ErrorsByType[errType]++
    s.SkippedRecords++
}

func (s *ImportStats) AddInvalidCourse(courseCode string) {
    s.InvalidCourses[courseCode]++
}

func (s *ImportStats) PrintSummary() {
    log.Printf("\nImport Statistics:")
    log.Printf("Total Records Processed: %d", s.TotalProcessed)
    log.Printf("Successfully Imported: %d", s.ValidRecords)
    log.Printf("Skipped Records: %d", s.SkippedRecords)
    
    if len(s.ErrorsByType) > 0 {
        log.Printf("\nErrors by Type:")
        for errType, count := range s.ErrorsByType {
            log.Printf("- %s: %d occurrences", errType, count)
        }
    }
    
    if len(s.InvalidCourses) > 0 {
        log.Printf("\nMost Common Invalid Course Codes:")
        // Convert map to slice for sorting
        type courseError struct {
            code  string
            count int
        }
        courses := make([]courseError, 0, len(s.InvalidCourses))
        for code, count := range s.InvalidCourses {
            courses = append(courses, courseError{code, count})
        }
        sort.Slice(courses, func(i, j int) bool {
            return courses[i].count > courses[j].count
        })
        
        // Show top 10 most common invalid courses
        for i := 0; i < min(10, len(courses)); i++ {
            log.Printf("- %s: %d occurrences", courses[i].code, courses[i].count)
        }
    }
}

func (d *DataImporter) DefaultColumnMappings() []ColumnMapping {
    // These are the columns we have in the CSV:
    // RegNumber,Lga,Gender,InstitutionCode,CourseCode,FacultyID,State,DateofBirth,Institution,Course
    return []ColumnMapping{
        // Map CSV columns in order
        {"RegNumber", "regnumber", transformString},
        {"MaritalStatus", "maritalstatus", func(s string) (interface{}, error) { return nil, nil }},
        {"Challenged", "challenged", func(s string) (interface{}, error) { return nil, nil }},
        {"Blind", "blind", func(s string) (interface{}, error) { return false, nil }},
        {"Deaf", "deaf", func(s string) (interface{}, error) { return false, nil }},
        {"ExamTown", "examtown", func(s string) (interface{}, error) { return nil, nil }},
        {"ExamCentre", "examcentre", func(s string) (interface{}, error) { return nil, nil }},
        {"ExamNo", "examno", func(s string) (interface{}, error) { return nil, nil }},
        {"Address", "address", func(s string) (interface{}, error) { return nil, nil }},
        {"NoOfSittings", "noofsittings", func(s string) (interface{}, error) { return nil, nil }},
        {"DateSaved", "datesaved", func(s string) (interface{}, error) { return nil, nil }},
        {"TimeSaved", "timesaved", func(s string) (interface{}, error) { return nil, nil }},
        {"MockCand", "mockcand", func(s string) (interface{}, error) { return false, nil }},
        {"MockState", "mockstate", func(s string) (interface{}, error) { return nil, nil }},
        {"MockTown", "mocktown", func(s string) (interface{}, error) { return nil, nil }},
        {"DateCreated", "datecreated", func(s string) (interface{}, error) { return nil, nil }},
        {"Email", "email", func(s string) (interface{}, error) { return nil, nil }},
        {"GSMNo", "gsmno", func(s string) (interface{}, error) { return nil, nil }},
        {"Surname", "surname", func(s string) (interface{}, error) { return nil, nil }},
        {"FirstName", "firstname", func(s string) (interface{}, error) { return nil, nil }},
        {"MiddleName", "middlename", func(s string) (interface{}, error) { return nil, nil }},
        {"DateofBirth", "dateofbirth", transformString},
        {"Gender", "gender", transformGender},
        {"State", "statecode", d.transformState},
        {"Subj1", "subj1", func(s string) (interface{}, error) { return nil, nil }},
        {"Score1", "score1", func(s string) (interface{}, error) { return nil, nil }},
        {"Subj2", "subj2", func(s string) (interface{}, error) { return nil, nil }},
        {"Score2", "score2", func(s string) (interface{}, error) { return nil, nil }},
        {"Subj3", "subj3", func(s string) (interface{}, error) { return nil, nil }},
        {"Score3", "score3", func(s string) (interface{}, error) { return nil, nil }},
        {"Subj4", "subj4", func(s string) (interface{}, error) { return nil, nil }},
        {"Score4", "score4", func(s string) (interface{}, error) { return nil, nil }},
        {"Aggregate", "aggregate", func(s string) (interface{}, error) { return nil, nil }},
        {"CourseCode", "app_course1", d.transformCourse},
        {"InstitutionCode", "inid", d.transformInstitution}, // Institution ID is varchar in DB
        {"Lga", "lg_id", transformInt},
        {"Year", "year", func(s string) (interface{}, error) { return d.config.Year, nil }},
        {"IsAdmitted", "is_admitted", d.transformAdmission},
    }
}

func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			if s1[i-1] == s2[j-1] {
				matrix[i][j] = matrix[i-1][j-1]
			} else {
				matrix[i][j] = min(
					matrix[i-1][j]+1,  // deletion
					matrix[i][j-1]+1,  // insertion
					matrix[i-1][j-1]+1, // substitution
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

func transformGender(s string) (interface{}, error) {
    if s == "" {
        return nil, nil
    }
    s = strings.ToUpper(strings.TrimSpace(s))
    switch s {
    case "M", "MALE":
        return "M", nil
    case "F", "FEMALE":
        return "F", nil
    default:
        return nil, fmt.Errorf("invalid gender value: %s", s)
    }
}

func (d *DataImporter) prepareInsertStatement(tx *sql.Tx) (*sql.Stmt, error) {
    columns := []string{
        "regnumber", "maritalstatus", "challenged", "blind", "deaf",
        "examtown", "examcentre", "examno", "address", "noofsittings",
        "datesaved", "timesaved", "mockcand", "mockstate", "mocktown",
        "datecreated", "email", "gsmno", "surname", "firstname",
        "middlename", "dateofbirth", "gender", "statecode",
        "subj1", "score1", "subj2", "score2", "subj3", "score3",
        "subj4", "score4", "aggregate", "app_course1", "inid",
        "lg_id", "year", "is_admitted"}

    placeholders := make([]string, len(columns))
    for i := range columns {
        placeholders[i] = fmt.Sprintf("$%d", i+1)
    }

    query := fmt.Sprintf(
        "INSERT INTO candidates (%s) VALUES (%s) ON CONFLICT (regnumber) DO UPDATE SET %s",
        strings.Join(columns, ", "),
        strings.Join(placeholders, ", "),
        buildUpdateClause(columns))

    return tx.Prepare(query)
}

func (d *DataImporter) transformRecord(headers []string, record []string) ([]interface{}, error) {
    values := make([]interface{}, 0, len(d.config.ColumnMappings))

    // Add values in the same order as the columns in prepareInsertStatement
    for _, mapping := range d.config.ColumnMappings {
        idx := getColumnIndex(headers, mapping.SourceColumn)
        if idx == -1 || idx >= len(record) {
            // For year field, use the year from config
            if mapping.DestinationColumn == "year" {
                values = append(values, d.config.Year)
                continue
            }
            values = append(values, nil)
            continue
        }

        value, err := mapping.TransformFunc(record[idx])
        if err != nil {
            return nil, fmt.Errorf("error transforming %s: %v", mapping.SourceColumn, err)
        }
        values = append(values, value)
    }

    return values, nil
}

func (d *DataImporter) insertRecord(record []interface{}) error {
    // Start a transaction for this record
    tx, err := d.db.Begin()
    if err != nil {
        return fmt.Errorf("error starting transaction: %v", err)
    }
    defer func() {
        if err != nil {
            tx.Rollback()
        }
    }()

    stmt, err := d.prepareInsertStatement(tx)
    if err != nil {
        return fmt.Errorf("error preparing statement: %v", err)
    }
    defer stmt.Close()

    _, err = stmt.Exec(record...)
    if err != nil {
        return fmt.Errorf("error inserting record: %v", err)
    }

    // Commit the transaction
    if err = tx.Commit(); err != nil {
        return fmt.Errorf("error committing transaction: %v", err)
    }

    return nil
}

func (d *DataImporter) printImportSummary(successCount, failedCount int, errors []error) {
    log.Printf("\nImport Summary:")
    log.Printf("Total Records Processed: %d", successCount+failedCount)
    log.Printf("Successfully Imported: %d (%.2f%%)", 
        successCount, 
        float64(successCount)/float64(successCount+failedCount)*100)
    log.Printf("Failed Records: %d (%.2f%%)", 
        failedCount,
        float64(failedCount)/float64(successCount+failedCount)*100)

    if len(errors) > 0 {
        log.Printf("\nSample of Import Errors (up to 10):")
        for i := 0; i < min(10, len(errors)); i++ {
            log.Printf("- %v", errors[i])
        }
    }
}

func buildUpdateClause(columns []string) string {
    updates := make([]string, 0, len(columns))
    for _, col := range columns {
        if col != "regnumber" { // Skip primary key in updates
            updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
        }
    }
    return strings.Join(updates, ", ")
}

// getColumnIndex returns the index of a column in headers
func getColumnIndex(headers []string, columnName string) int {
    for i, header := range headers {
        // Normalize both strings for comparison
        normalizedHeader := strings.ToLower(strings.TrimSpace(header))
        normalizedColumn := strings.ToLower(strings.TrimSpace(columnName))
        
        // Try exact match first
        if normalizedHeader == normalizedColumn {
            return i
        }
        
        // Try with common variations
        headerNoSpace := strings.ReplaceAll(normalizedHeader, " ", "")
        columnNoSpace := strings.ReplaceAll(normalizedColumn, " ", "")
        if headerNoSpace == columnNoSpace {
            return i
        }
        
        headerNoUnderscore := strings.ReplaceAll(normalizedHeader, "_", "")
        columnNoUnderscore := strings.ReplaceAll(normalizedColumn, "_", "")
        if headerNoUnderscore == columnNoUnderscore {
            return i
        }
    }
    return -1
}

// max returns the maximum of two integers
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func ImportCourses(ctx context.Context, db *sql.DB, filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return fmt.Errorf("error opening file: %w", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
    headers, err := reader.Read()
    if err != nil {
        return fmt.Errorf("error reading headers: %w", err)
    }

    // Find column indices
    codeIdx := -1
    nameIdx := -1
    for i, header := range headers {
        switch strings.ToLower(strings.TrimSpace(header)) {
        case "course_code":
            codeIdx = i
        case "course_name":
            nameIdx = i
        }
    }

    if codeIdx == -1 || nameIdx == -1 {
        return fmt.Errorf("required columns 'course_code' and 'course_name' not found in CSV")
    }

    courseMapper := NewCourseMapper(db)
    var updated, skipped int

    // Start a transaction
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to start transaction: %w", err)
    }
    defer tx.Rollback()

    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            return fmt.Errorf("error reading record: %w", err)
        }

        courseCode := strings.TrimSpace(record[codeIdx])
        courseName := strings.TrimSpace(record[nameIdx])

        if courseCode == "" || courseName == "" {
            skipped++
            continue
        }

        if err := courseMapper.UpsertCourse(courseCode, courseName); err != nil {
            return fmt.Errorf("error upserting course %s: %w", courseCode, err)
        }
        updated++
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    log.Printf("Course import completed: %d updated, %d skipped", updated, skipped)
    return nil
}

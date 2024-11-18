package main

import (
    "bufio"
    "context"
    "database/sql"
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "syscall"
    "time"

    "github.com/fatih/color"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
    "github.com/olekukonko/tablewriter"
    "github.com/nonsonwune/spk2_db/nlquery"
    "github.com/nonsonwune/spk2_db/importer"
    "github.com/nonsonwune/spk2_db/migrations"
)

// Config holds application configuration
type Config struct {
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
}

func loadConfig() (*Config, error) {
    if err := godotenv.Load(); err != nil {
        return nil, fmt.Errorf("error loading .env file: %w", err)
    }

    return &Config{
        DBHost:     os.Getenv("DB_HOST"),
        DBPort:     os.Getenv("DB_PORT"),
        DBUser:     os.Getenv("DB_USER"),
        DBPassword: os.Getenv("DB_PASSWORD"),
        DBName:     os.Getenv("DB_NAME"),
    }, nil
}

func connectDB(cfg *Config) (*sql.DB, error) {
    psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

    db, err := sql.Open("postgres", psqlInfo)
    if err != nil {
        return nil, fmt.Errorf("error opening database: %w", err)
    }

    // Set connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("error connecting to database: %w", err)
    }

    return db, nil
}

func main() {
    // Load configuration
    cfg, err := loadConfig()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Connect to database
    db, err := connectDB(cfg)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Initialize database schema
    if err := migrations.InitSchema(db); err != nil {
        log.Printf("Warning: Error initializing schema: %v", err)
    }

    // Setup signal handling for graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

    go func() {
        sig := <-signalChan
        log.Printf("Received signal: %v", sig)
        cancel()
    }()

    // Start menu loop
    menuLoop(ctx, db)
}

func menuLoop(ctx context.Context, db *sql.DB) {
    for {
        select {
        case <-ctx.Done():
            color.Yellow("\nShutting down gracefully...")
            return
        default:
            displayMenu()
            choice := readChoice()

            if err := handleMenuChoice(ctx, db, choice); err != nil {
                if err == errExit {
                    color.Green("Thank you for using JAMB Candidates Management System!")
                    return
                }
                color.Red("Error: %v", err)
            }
        }
    }
}

var errExit = fmt.Errorf("exit requested")

func handleMenuChoice(ctx context.Context, db *sql.DB, choice string) error {
    switch choice {
    case "1":
        return searchCandidates(ctx, db)
    case "2":
        return displayTopPerformers(ctx, db)
    case "3":
        return displayGenderStats(ctx, db)
    case "4":
        return displayStateDistribution(ctx, db)
    case "5":
        return displaySubjectStats(ctx, db)
    case "6":
        return displayAggregateDistribution(ctx, db)
    case "7":
        return displayCourseAnalysis(ctx, db)
    case "8":
        return displayInstitutionStats(ctx, db)
    case "9":
        return displayFacultyPerformance(ctx, db)
    case "10":
        return displayGeographicAnalysis(ctx, db)
    case "11":
        return displayYearComparison(ctx, db)
    case "12":
        return displayAdmissionTrends(ctx, db)
    case "13":
        return handleCandidateImport(ctx, db)
    case "14":
        return handleAnalyzeFailedImports(ctx, db)
    case "15":
        return displayPerformanceMetrics(ctx, db)
    case "16":
        return displayInstitutionRanking(ctx, db)
    case "17":
        return displaySubjectCorrelation(ctx, db)
    case "18":
        return displayRegionalPerformance(ctx, db)
    case "19":
        return displayCourseCompetitiveness(ctx, db)
    case "20":
        return handleNaturalLanguageQuery(ctx, db)
    case "0":
        return errExit
    default:
        return fmt.Errorf("invalid choice")
    }
}

func displayMenu() {
    color.Green("\nJAMB Database Analysis System")
    color.Green("============================")
    fmt.Println("1. Search Candidates")
    fmt.Println("2. Top Performers")
    fmt.Println("3. Gender Statistics")
    fmt.Println("4. State Distribution")
    fmt.Println("5. Subject Statistics")
    fmt.Println("6. Aggregate Score Distribution")
    fmt.Println("7. Course Analysis")
    fmt.Println("8. Institution Statistics")
    fmt.Println("9. Faculty Performance")
    fmt.Println("10. Geographic Analysis")
    fmt.Println("11. Year-over-Year Comparison")
    fmt.Println("12. Admission Trends")
    fmt.Println("13. Import Candidates")
    fmt.Println("14. Analyze Failed Imports")
    fmt.Println("15. Performance Metrics")
    fmt.Println("16. Institution Ranking")
    fmt.Println("17. Subject Correlation")
    fmt.Println("18. Regional Performance")
    fmt.Println("19. Course Competitiveness")
    fmt.Println("20. Natural Language Query")
    fmt.Println("0. Exit")
    fmt.Print("\nEnter your choice: ")
}

func searchCandidates(ctx context.Context, db *sql.DB) error {
    var searchTerm string
    fmt.Print("Enter registration number or surname to search: ")
    scanner := bufio.NewScanner(os.Stdin)
    if scanner.Scan() {
        searchTerm = scanner.Text()
    }

    query := `
        SELECT regnumber, surname, firstname, gender, aggregate 
        FROM candidate 
        WHERE regnumber LIKE $1 OR LOWER(surname) LIKE LOWER($1)
        LIMIT 10
    `

    rows, err := db.QueryContext(ctx, query, "%"+searchTerm+"%")
    if err != nil {
        log.Printf("Error searching candidates: %v", err)
        return err
    }
    defer rows.Close()

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Reg Number", "Surname", "First Name", "Gender", "Aggregate"})

    for rows.Next() {
        var reg, surname, firstname, gender sql.NullString
        var aggregate sql.NullInt64

        err := rows.Scan(&reg, &surname, &firstname, &gender, &aggregate)
        if err != nil {
            log.Printf("Error scanning row: %v", err)
            continue
        }

        table.Append([]string{
            getString(reg),
            getString(surname),
            getString(firstname),
            getString(gender),
            fmt.Sprintf("%d", getInt64(aggregate)),
        })
    }

    table.Render()
    return nil
}

func displayTopPerformers(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT regnumber, surname, firstname, aggregate 
        FROM candidate 
        WHERE aggregate IS NOT NULL 
        ORDER BY aggregate DESC 
        LIMIT 10
    `

    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting top performers: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nTop 10 Performers")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Rank", "Reg Number", "Name", "Aggregate"})

    rank := 1
    for rows.Next() {
        var reg, surname, firstname sql.NullString
        var aggregate sql.NullInt64

        err := rows.Scan(&reg, &surname, &firstname, &aggregate)
        if err != nil {
            continue
        }

        name := fmt.Sprintf("%s %s", getString(surname), getString(firstname))
        table.Append([]string{
            fmt.Sprintf("%d", rank),
            getString(reg),
            name,
            fmt.Sprintf("%d", getInt64(aggregate)),
        })
        rank++
    }

    table.Render()
    return nil
}

func displayGenderStats(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT gender, COUNT(*) as count 
        FROM candidate 
        WHERE gender IS NOT NULL 
        GROUP BY gender
    `

    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting gender stats: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nGender Distribution")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Gender", "Count"})

    for rows.Next() {
        var gender string
        var count int

        err := rows.Scan(&gender, &count)
        if err != nil {
            continue
        }

        table.Append([]string{
            gender,
            fmt.Sprintf("%d", count),
        })
    }

    table.Render()
    return nil
}

func displayStateDistribution(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT s.st_name, COUNT(c.*) as count 
        FROM candidate c
        JOIN state s ON c.statecode = s.st_id
        GROUP BY s.st_name 
        ORDER BY count DESC
        LIMIT 10
    `

    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting state distribution: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nTop 10 States by Number of Candidates")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"State", "Number of Candidates"})

    for rows.Next() {
        var state string
        var count int

        err := rows.Scan(&state, &count)
        if err != nil {
            continue
        }

        table.Append([]string{
            state,
            fmt.Sprintf("%d", count),
        })
    }

    table.Render()
    return nil
}

func displaySubjectStats(ctx context.Context, db *sql.DB) error {
    query := `
        WITH RankedSubjects AS (
            SELECT 
                s.su_name,
                cs.score,
                COUNT(*) as count,
                RANK() OVER (PARTITION BY cs.cand_reg_number ORDER BY cs.score DESC) as score_rank
            FROM candidate c
            JOIN candidate_scores cs ON c.regnumber = cs.cand_reg_number AND c.year = cs.year
            JOIN subject s ON cs.subject_id = s.su_id
            WHERE c.year = (SELECT MAX(year) FROM candidate)
            GROUP BY s.su_name, cs.score, cs.cand_reg_number
        )
        SELECT 
            su_name,
            COUNT(*) as total_candidates,
            ROUND(AVG(score)::numeric, 2) as avg_score
        FROM RankedSubjects
        WHERE score_rank = 1
        GROUP BY su_name
        ORDER BY total_candidates DESC
        LIMIT 5;
    `

    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting subject stats: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nAverage Scores by Subject")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Subject", "Total Candidates", "Average Score"})

    for rows.Next() {
        var subject string
        var totalCandidates int
        var avgScore float64

        err := rows.Scan(&subject, &totalCandidates, &avgScore)
        if err != nil {
            continue
        }

        table.Append([]string{
            subject,
            fmt.Sprintf("%d", totalCandidates),
            fmt.Sprintf("%.2f", avgScore),
        })
    }

    table.Render()
    return nil
}

func displayAggregateDistribution(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT 
            CASE 
                WHEN aggregate >= 300 THEN '300+'
                WHEN aggregate >= 250 THEN '250-299'
                WHEN aggregate >= 200 THEN '200-249'
                WHEN aggregate >= 150 THEN '150-199'
                ELSE 'Below 150'
            END as range,
            COUNT(*) as count
        FROM candidate
        WHERE aggregate IS NOT NULL
        GROUP BY range
        ORDER BY range DESC
    `

    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting aggregate distribution: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nAggregate Score Distribution")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Score Range", "Number of Candidates"})

    for rows.Next() {
        var scoreRange string
        var count int

        err := rows.Scan(&scoreRange, &count)
        if err != nil {
            continue
        }

        table.Append([]string{
            scoreRange,
            fmt.Sprintf("%d", count),
        })
    }

    table.Render()
    return nil
}

func displayCourseAnalysis(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT c.course_name, COUNT(ca.regnumber) as applicants,
               ROUND(AVG(ca.aggregate)::numeric, 2) as avg_score,
               f.name as faculty
        FROM course c
        LEFT JOIN candidate ca ON c.course_code = ca.app_course1
        LEFT JOIN faculty f ON c.faculty_id = f.id
        GROUP BY c.course_name, f.name
        ORDER BY applicants DESC
        LIMIT 15
    `
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting course analysis: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nTop 15 Courses by Number of Applicants")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Course", "Faculty", "Applicants", "Average Score"})

    for rows.Next() {
        var course, faculty string
        var applicants int
        var avgScore float64

        err := rows.Scan(&course, &applicants, &avgScore, &faculty)
        if err != nil {
            continue
        }

        table.Append([]string{
            course,
            faculty,
            fmt.Sprintf("%d", applicants),
            fmt.Sprintf("%.2f", avgScore),
        })
    }

    table.Render()
    return nil
}

func displayInstitutionStats(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT i.inname, COUNT(c.regnumber) as applicants,
               ROUND(AVG(c.aggregate)::numeric, 2) as avg_score,
               it.name as institution_type
        FROM institution i
        LEFT JOIN candidate c ON i.inid = c.inid
        LEFT JOIN institution_type it ON i.institution_type_id = it.id
        GROUP BY i.inname, it.name
        ORDER BY applicants DESC
        LIMIT 15
    `
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting institution stats: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nTop 15 Institutions by Number of Applicants")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Institution", "Type", "Applicants", "Average Score"})

    for rows.Next() {
        var institution, instType string
        var applicants int
        var avgScore float64

        err := rows.Scan(&institution, &applicants, &avgScore, &instType)
        if err != nil {
            continue
        }

        table.Append([]string{
            institution,
            instType,
            fmt.Sprintf("%d", applicants),
            fmt.Sprintf("%.2f", avgScore),
        })
    }

    table.Render()
    return nil
}

func displayFacultyPerformance(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT f.name, COUNT(c.regnumber) as applicants,
               ROUND(AVG(c.aggregate)::numeric, 2) as avg_score
        FROM faculty f
        JOIN course co ON f.id = co.faculty_id
        LEFT JOIN candidate c ON co.course_code = c.app_course1
        GROUP BY f.name
        ORDER BY avg_score DESC
    `
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting faculty performance: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nFaculty Performance Analysis")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Faculty", "Total Applicants", "Average Score"})

    for rows.Next() {
        var faculty string
        var applicants int
        var avgScore float64

        err := rows.Scan(&faculty, &applicants, &avgScore)
        if err != nil {
            continue
        }

        table.Append([]string{
            faculty,
            fmt.Sprintf("%d", applicants),
            fmt.Sprintf("%.2f", avgScore),
        })
    }

    table.Render()
    return nil
}

func displayGeographicAnalysis(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT s.st_name as state, l.lg_name as lga,
               COUNT(c.regnumber) as candidates,
               ROUND(AVG(c.aggregate)::numeric, 2) as avg_score
        FROM state s
        JOIN lga l ON s.st_id = l.state_id
        JOIN candidate c ON l.lg_id = c.lgaid
        GROUP BY s.st_name, l.lg_name
        HAVING COUNT(c.regnumber) > 1000
        ORDER BY candidates DESC
        LIMIT 15
    `
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting geographic analysis: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nTop 15 LGAs by Number of Candidates")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"State", "LGA", "Candidates", "Average Score"})

    for rows.Next() {
        var state, lga string
        var candidates int
        var avgScore float64

        err := rows.Scan(&state, &lga, &candidates, &avgScore)
        if err != nil {
            continue
        }

        table.Append([]string{
            state,
            lga,
            fmt.Sprintf("%d", candidates),
            fmt.Sprintf("%.2f", avgScore),
        })
    }

    table.Render()
    return nil
}

func displayYearComparison(ctx context.Context, db *sql.DB) error {
    query := `
        SELECT year,
               COUNT(*) as total_candidates,
               ROUND(AVG(aggregate)::numeric, 2) as avg_score,
               COUNT(CASE WHEN gender = 'F' THEN 1 END) as female_candidates,
               COUNT(CASE WHEN gender = 'M' THEN 1 END) as male_candidates
        FROM candidate
        GROUP BY year
        ORDER BY year
    `
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting year comparison: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nYear-wise Statistics")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Year", "Total Candidates", "Average Score", "Female", "Male"})

    for rows.Next() {
        var year, totalCandidates, femaleCandidates, maleCandidates int
        var avgScore float64

        err := rows.Scan(&year, &totalCandidates, &avgScore, &femaleCandidates, &maleCandidates)
        if err != nil {
            continue
        }

        table.Append([]string{
            fmt.Sprintf("%d", year),
            fmt.Sprintf("%d", totalCandidates),
            fmt.Sprintf("%.2f", avgScore),
            fmt.Sprintf("%d", femaleCandidates),
            fmt.Sprintf("%d", maleCandidates),
        })
    }

    table.Render()
    return nil
}

func displayAdmissionTrends(ctx context.Context, db *sql.DB) error {
    query := `
        WITH course_stats AS (
            SELECT 
                c.course_name,
                COUNT(*) as applicants,
                PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY ca.aggregate) as cutoff_score
            FROM course c
            JOIN candidate ca ON c.course_code = ca.app_course1
            GROUP BY c.course_name
            HAVING COUNT(*) > 100
        )
        SELECT name,
               applicants,
               ROUND(cutoff_score::numeric, 2) as cutoff_score
        FROM course_stats
        ORDER BY applicants DESC
        LIMIT 15
    `
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Error getting admission trends: %v", err)
        return err
    }
    defer rows.Close()

    color.Yellow("\nAdmission Trends (Top 15 Courses)")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Course", "Total Applicants", "Estimated Cutoff Score"})

    for rows.Next() {
        var course string
        var applicants int
        var cutoffScore float64

        err := rows.Scan(&course, &applicants, &cutoffScore)
        if err != nil {
            continue
        }

        table.Append([]string{
            course,
            fmt.Sprintf("%d", applicants),
            fmt.Sprintf("%.2f", cutoffScore),
        })
    }

    table.Render()
    return nil
}

func readChoice() string {
    var input string
    fmt.Scanln(&input)
    return strings.TrimSpace(input)
}

func readString() string {
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Scan()
    return strings.TrimSpace(scanner.Text())
}

func readInt() int {
    var input string
    fmt.Scanln(&input)
    i, _ := strconv.Atoi(input)
    return i
}

// Helper functions
func getString(s sql.NullString) string {
    if s.Valid {
        return s.String
    }
    return "N/A"
}

func getInt64(i sql.NullInt64) int64 {
    if i.Valid {
        return i.Int64
    }
    return 0
}

func handleCandidateImport(ctx context.Context, db *sql.DB) error {
    // Check if context is already cancelled
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    fmt.Print("Enter the CSV file path: ")
    filename := readString()

    // Check context after user input
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    fmt.Print("Enter the year for the data (e.g., 2023): ")
    year := readInt()

    // Check context after user input
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    fmt.Print("Is this admission data? (y/n): ")
    isAdmission := strings.ToLower(readString()) == "y"

    // Check context after user input
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    workerCount := 4 // default value
    if envWorkerCount := os.Getenv("WORKER_COUNT"); envWorkerCount != "" {
        if count, err := strconv.Atoi(envWorkerCount); err == nil && count > 0 {
            workerCount = count
        }
    }

    fmt.Printf("\nUsing %d workers for parallel processing\n", workerCount)

    fmt.Printf("\nReady to import data from %s for year %d\n", filename, year)
    if isAdmission {
        fmt.Println("This will be imported as admission data")
    }
    fmt.Print("Proceed with import? (y/n): ")

    if strings.ToLower(readString()) == "y" {
        // Check context again before starting the import
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // Open the CSV file
        file, err := os.Open(filename)
        if err != nil {
            color.Red("Error opening file: %v", err)
            return fmt.Errorf("error opening file: %w", err)
        }
        defer file.Close()

        // Create a buffered reader for better performance
        bufferedReader := bufio.NewReader(file)
        reader := csv.NewReader(bufferedReader)

        config := importer.ImportConfig{
            Year:        year,
            SourceFile:  filename,
            IsAdmission: isAdmission,
            BatchSize:   1000,
            WorkerCount: workerCount,
        }

        // Create a child context with timeout for the import operation
        importCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
        defer cancel()

        // Create a progress indicator
        go func() {
            ticker := time.NewTicker(5 * time.Second)
            defer ticker.Stop()

            for {
                select {
                case <-importCtx.Done():
                    return
                case <-ticker.C:
                    fmt.Print(".")
                }
            }
        }()

        fmt.Print("\nImporting data")

        // Pass the context to ImportData
        if err := importer.ImportData(importCtx, db, config, reader); err != nil {
            fmt.Println() // New line after progress dots
            switch {
            case err == context.DeadlineExceeded:
                color.Red("Import timed out after 30 minutes")
                return fmt.Errorf("import timed out: %w", err)
            case err == context.Canceled:
                color.Yellow("Import was cancelled")
                return fmt.Errorf("import cancelled: %w", err)
            default:
                color.Red("Error importing data: %v", err)
                return fmt.Errorf("import error: %w", err)
            }
        }
        
        fmt.Println() // New line after progress dots
        color.Green("Import completed successfully!")
    } else {
        fmt.Println("Import cancelled.")
    }
    return nil
}

func handleAnalyzeFailedImports(ctx context.Context, db *sql.DB) error {
    // Use context for database queries
    query := `
        SELECT error_message, COUNT(*) as count
        FROM import_errors
        GROUP BY error_message
        ORDER BY count DESC
        LIMIT 10
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        color.Red("Error analyzing failed imports: %v", err)
        return err
    }
    defer rows.Close()

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Error Message", "Count"})

    for rows.Next() {
        var message string
        var count int
        if err := rows.Scan(&message, &count); err != nil {
            color.Red("Error scanning row: %v", err)
            continue
        }
        table.Append([]string{
            message,
            strconv.Itoa(count),
        })
    }

    if err = rows.Err(); err != nil {
        color.Red("Error iterating rows: %v", err)
        return err
    }

    color.Cyan("\nFailed Import Analysis")
    table.Render()
    return nil
}

func displayPerformanceMetrics(ctx context.Context, db *sql.DB) error {
    query := `
        WITH ScoreStats AS (
            SELECT 
                year,
                COUNT(*) as total_candidates,
                AVG(NULLIF(aggregate, 0)) as avg_score,
                PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY NULLIF(aggregate, 0)) as median_score,
                STDDEV(NULLIF(aggregate, 0)) as std_dev
            FROM candidate 
            WHERE aggregate IS NOT NULL AND aggregate > 0
            GROUP BY year
        )
        SELECT 
            year,
            total_candidates,
            COALESCE(ROUND(avg_score::numeric, 2), 0) as average_score,
            COALESCE(ROUND(median_score::numeric, 2), 0) as median_score,
            COALESCE(ROUND(std_dev::numeric, 2), 0) as standard_deviation
        FROM ScoreStats
        ORDER BY year DESC;
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        color.Red("Error fetching performance metrics: %v", err)
        return err
    }
    defer rows.Close()

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Year", "Total Candidates", "Average Score", "Median Score", "Std Deviation"})

    for rows.Next() {
        var year, totalCandidates int
        var avgScore, medianScore, stdDev float64
        
        if err := rows.Scan(&year, &totalCandidates, &avgScore, &medianScore, &stdDev); err != nil {
            color.Red("Error scanning row: %v", err)
            continue
        }
        
        table.Append([]string{
            strconv.Itoa(year),
            strconv.Itoa(totalCandidates),
            fmt.Sprintf("%.2f", avgScore),
            fmt.Sprintf("%.2f", medianScore),
            fmt.Sprintf("%.2f", stdDev),
        })
    }

    color.Cyan("\nPerformance Metrics Analysis")
    table.Render()
    return nil
}

func displayInstitutionRanking(ctx context.Context, db *sql.DB) error {
    query := `
        WITH AdmissionStats AS (
            SELECT 
                i.inname as institution_name,
                i.inabv as abbreviation,
                COUNT(c.regnumber) as total_applicants,
                COUNT(CASE WHEN c.is_admitted = true THEN 1 END) as admitted_count,
                AVG(NULLIF(c.aggregate, 0)) as avg_score
            FROM institution i
            LEFT JOIN candidate c ON i.inid = c.inid
            WHERE c.year = (SELECT MAX(year) FROM candidate)
                AND c.aggregate IS NOT NULL 
                AND c.aggregate > 0
            GROUP BY i.inname, i.inabv
            HAVING COUNT(c.regnumber) > 100
        )
        SELECT 
            institution_name,
            abbreviation,
            total_applicants,
            admitted_count,
            COALESCE(ROUND(avg_score::numeric, 2), 0) as average_score,
            ROUND((admitted_count::float / total_applicants * 100)::numeric, 2) as admission_rate
        FROM AdmissionStats
        ORDER BY avg_score DESC
        LIMIT 20;
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        color.Red("Error fetching institution rankings: %v", err)
        return err
    }
    defer rows.Close()

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Institution", "Abbrev", "Total Applicants", "Admitted", "Avg Score", "Admission Rate (%)"})

    for rows.Next() {
        var name, abbrev string
        var totalApplicants, admitted int
        var avgScore, admissionRate float64
        
        if err := rows.Scan(&name, &abbrev, &totalApplicants, &admitted, &avgScore, &admissionRate); err != nil {
            color.Red("Error scanning row: %v", err)
            continue
        }
        
        table.Append([]string{
            name,
            abbrev,
            strconv.Itoa(totalApplicants),
            strconv.Itoa(admitted),
            fmt.Sprintf("%.2f", avgScore),
            fmt.Sprintf("%.2f%%", admissionRate),
        })
    }

    color.Cyan("\nTop 20 Institutions by Average Score (Latest Year)")
    table.Render()
    return nil
}

func displaySubjectCorrelation(ctx context.Context, db *sql.DB) error {
    query := `
        WITH EnglishScores AS (
            SELECT 
                cs.cand_reg_number,
                cs.score as english_score
            FROM candidate_scores cs
            JOIN subject s ON cs.subject_id = s.su_id
            WHERE s.su_name = 'USE OF ENGLISH'
            AND cs.year = (SELECT MAX(year) FROM candidate)
        ),
        OtherSubjectScores AS (
            SELECT 
                cs.cand_reg_number,
                s.su_name as subject_name,
                cs.score as subject_score
            FROM candidate_scores cs
            JOIN subject s ON cs.subject_id = s.su_id
            WHERE s.su_name != 'USE OF ENGLISH'
            AND cs.year = (SELECT MAX(year) FROM candidate)
        ),
        SubjectCorrelations AS (
            SELECT 
                o.subject_name,
                COUNT(*) as sample_size,
                CORR(e.english_score, o.subject_score) as correlation,
                AVG(e.english_score) as avg_english,
                AVG(o.subject_score) as avg_subject,
                STDDEV(e.english_score) as stddev_english,
                STDDEV(o.subject_score) as stddev_subject
            FROM EnglishScores e
            JOIN OtherSubjectScores o ON e.cand_reg_number = o.cand_reg_number
            GROUP BY o.subject_name
            HAVING COUNT(*) >= 1000  -- Ensure statistical significance
            AND STDDEV(e.english_score) > 0 
            AND STDDEV(o.subject_score) > 0
        )
        SELECT 
            'USE OF ENGLISH' as subject1,
            subject_name as subject2,
            ROUND(correlation::numeric, 3) as correlation,
            sample_size,
            ROUND(avg_english::numeric, 2) as avg_score1,
            ROUND(avg_subject::numeric, 2) as avg_score2,
            ROUND(stddev_english::numeric, 2) as stddev1,
            ROUND(stddev_subject::numeric, 2) as stddev2
        FROM SubjectCorrelations
        WHERE correlation IS NOT NULL
        ORDER BY ABS(correlation) DESC;
    `

    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        color.Red("Error fetching subject correlations: %v", err)
        return err
    }
    defer rows.Close()

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{
        "Subject 1", 
        "Subject 2", 
        "Correlation", 
        "Sample Size",
        "Avg Score 1", 
        "Avg Score 2", 
        "StdDev 1", 
        "StdDev 2",
    })

    hasRows := false
    for rows.Next() {
        hasRows = true
        var (
            subject1, subject2 string
            correlation        float64
            sampleSize        int
            avgScore1, avgScore2, stdDev1, stdDev2 float64
        )

        if err := rows.Scan(&subject1, &subject2, &correlation, &sampleSize,
            &avgScore1, &avgScore2, &stdDev1, &stdDev2); err != nil {
            color.Red("Error scanning row: %v", err)
            continue
        }

        table.Append([]string{
            subject1,
            subject2,
            fmt.Sprintf("%.3f", correlation),
            fmt.Sprintf("%d", sampleSize),
            fmt.Sprintf("%.2f", avgScore1),
            fmt.Sprintf("%.2f", avgScore2),
            fmt.Sprintf("%.2f", stdDev1),
            fmt.Sprintf("%.2f", stdDev2),
        })
    }

    color.Cyan("\nSubject Score Correlations (Latest Year)\n")
    if !hasRows {
        color.Yellow("No significant correlations found between subjects.")
    } else {
        table.Render()
    }

    return nil
}

func displayRegionalPerformance(ctx context.Context, db *sql.DB) error {
    query := `
        WITH RegionalStats AS (
            SELECT 
                s.st_name as state_name,
                COUNT(c.regnumber) as total_candidates,
                AVG(NULLIF(c.aggregate, 0)) as avg_score,
                COUNT(CASE WHEN c.is_admitted = true THEN 1 END) as admitted_count,
                COUNT(CASE WHEN c.gender = 'F' THEN 1 END) as female_count
            FROM candidate c
            JOIN state s ON c.statecode = s.st_id
            WHERE c.year = (SELECT MAX(year) FROM candidate)
                AND c.aggregate IS NOT NULL 
                AND c.aggregate > 0
            GROUP BY s.st_name
        )
        SELECT 
            state_name,
            total_candidates,
            COALESCE(ROUND(avg_score::numeric, 2), 0) as average_score,
            admitted_count,
            ROUND((female_count::float / total_candidates * 100)::numeric, 2) as female_percentage
        FROM RegionalStats
        ORDER BY total_candidates DESC;
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        color.Red("Error fetching regional performance: %v", err)
        return err
    }
    defer rows.Close()

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"State", "Total Candidates", "Avg Score", "Admitted", "Female %"})

    for rows.Next() {
        var stateName string
        var totalCandidates, admitted int
        var avgScore, femalePercentage float64
        
        if err := rows.Scan(&stateName, &totalCandidates, &avgScore, &admitted, &femalePercentage); err != nil {
            color.Red("Error scanning row: %v", err)
            continue
        }
        
        table.Append([]string{
            stateName,
            strconv.Itoa(totalCandidates),
            fmt.Sprintf("%.2f", avgScore),
            strconv.Itoa(admitted),
            fmt.Sprintf("%.2f%%", femalePercentage),
        })
    }

    color.Cyan("\nRegional Performance Analysis (Latest Year)")
    table.Render()
    return nil
}

func displayCourseCompetitiveness(ctx context.Context, db *sql.DB) error {
    query := `
        WITH CourseStats AS (
            SELECT 
                c.app_course1 as course_code,
                co.course_name as course_name,
                COUNT(c.regnumber) as total_applicants,
                MIN(NULLIF(c.aggregate, 0)) as min_score,
                MAX(NULLIF(c.aggregate, 0)) as max_score,
                AVG(NULLIF(c.aggregate, 0)) as avg_score,
                COUNT(CASE WHEN c.is_admitted = true THEN 1 END) as admitted_count
            FROM candidate c
            JOIN course co ON c.app_course1 = co.course_code
            WHERE c.year = (SELECT MAX(year) FROM candidate)
                AND c.aggregate IS NOT NULL 
                AND c.aggregate > 0
            GROUP BY c.app_course1, co.course_name
            HAVING COUNT(c.regnumber) > 50
        )
        SELECT 
            course_name,
            total_applicants,
            COALESCE(ROUND(min_score::numeric, 2), 0) as minimum_score,
            COALESCE(ROUND(max_score::numeric, 2), 0) as maximum_score,
            COALESCE(ROUND(avg_score::numeric, 2), 0) as average_score,
            ROUND((admitted_count::float / total_applicants * 100)::numeric, 2) as admission_rate
        FROM CourseStats
        ORDER BY avg_score DESC
        LIMIT 20;
    `
    
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        color.Red("Error fetching course competitiveness: %v", err)
        return err
    }
    defer rows.Close()

    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Course", "Applicants", "Min Score", "Max Score", "Avg Score", "Admission Rate (%)"})

    for rows.Next() {
        var courseName string
        var totalApplicants int
        var minScore, maxScore, avgScore, admissionRate float64
        
        if err := rows.Scan(&courseName, &totalApplicants, &minScore, &maxScore, &avgScore, &admissionRate); err != nil {
            color.Red("Error scanning row: %v", err)
            continue
        }
        
        table.Append([]string{
            courseName,
            strconv.Itoa(totalApplicants),
            fmt.Sprintf("%.2f", minScore),
            fmt.Sprintf("%.2f", maxScore),
            fmt.Sprintf("%.2f", avgScore),
            fmt.Sprintf("%.2f%%", admissionRate),
        })
    }

    color.Cyan("\nTop 20 Most Competitive Courses (Latest Year)")
    table.Render()
    return nil
}

func handleNaturalLanguageQuery(ctx context.Context, db *sql.DB) error {
    // Initialize database configuration
    dbConfig := map[string]string{
        "host":     os.Getenv("DB_HOST"),
        "port":     os.Getenv("DB_PORT"),
        "user":     os.Getenv("DB_USER"),
        "password": os.Getenv("DB_PASSWORD"),
        "dbname":   os.Getenv("DB_NAME"),
    }

    // Initialize NL Query Engine
    engine, err := nlquery.NewNLQueryEngine(dbConfig)
    if err != nil {
        return fmt.Errorf("error initializing NL Query Engine: %v", err)
    }
    defer engine.Close()

    fmt.Println("\nNatural Language Query")
    fmt.Println("=====================")
    fmt.Println("Enter your question (or 'exit' to return to menu):")

    for {
        fmt.Print("\nQuery: ")
        query := readString()
        
        if strings.ToLower(query) == "exit" {
            return nil
        }

        if err := engine.ProcessQuery(ctx, query); err != nil {
            fmt.Printf("Error processing query: %v\n", err)
        }
    }
}

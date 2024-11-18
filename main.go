package main

import (
    "bufio"
    "database/sql"
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"

    "github.com/fatih/color"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
    "github.com/olekukonko/tablewriter"
    "github.com/nonsonwune/spk2_db/importer"
    "github.com/nonsonwune/spk2_db/migrations"
)

func init() {
    // Load .env file
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
}

func main() {
    // Connect to database using environment variables
    psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"))

    db, err := sql.Open("postgres", psqlInfo)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Test connection
    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }

    // Initialize database schema
    if err := migrations.InitSchema(db); err != nil {
        log.Printf("Warning: Error initializing schema: %v", err)
    }

    for {
        displayMenu()
        choice := readChoice()

        switch choice {
        case "1":
            searchCandidates(db)
        case "2":
            displayTopPerformers(db)
        case "3":
            displayGenderStats(db)
        case "4":
            displayStateDistribution(db)
        case "5":
            displaySubjectStats(db)
        case "6":
            displayAggregateDistribution(db)
        case "7":
            displayCourseAnalysis(db)
        case "8":
            displayInstitutionStats(db)
        case "9":
            displayFacultyPerformance(db)
        case "10":
            displayGeographicAnalysis(db)
        case "11":
            displayYearComparison(db)
        case "12":
            displayAdmissionTrends(db)
        case "13":
            handleCandidateImport(db)
        case "14":
            handleAnalyzeFailedImports(db)
        case "15":
            displayPerformanceMetrics(db)
        case "16":
            displayInstitutionRanking(db)
        case "17":
            displaySubjectCorrelation(db)
        case "18":
            displayRegionalPerformance(db)
        case "19":
            displayCourseCompetitiveness(db)
        case "20":
            color.Green("Thank you for using JAMB Candidates Management System!")
            return
        default:
            color.Red("Invalid choice. Please try again.")
        }
    }
}

func displayMenu() {
    color.Cyan("\n=== JAMB Candidates Management System ===")
    fmt.Println("1. Search Candidates")
    fmt.Println("2. View Top Performers")
    fmt.Println("3. Gender Statistics")
    fmt.Println("4. State Distribution")
    fmt.Println("5. Subject Statistics")
    fmt.Println("6. Aggregate Score Distribution")
    fmt.Println("7. Course Analysis")
    fmt.Println("8. Institution Statistics")
    fmt.Println("9. Faculty Performance")
    fmt.Println("10. Geographic Analysis")
    fmt.Println("11. Year Comparison")
    fmt.Println("12. Admission Trends")
    fmt.Println("13. Import Candidates Data")
    fmt.Println("14. Analyze Failed Imports")
    fmt.Println("15. Performance Metrics")
    fmt.Println("16. Institution Ranking")
    fmt.Println("17. Subject Correlation Analysis")
    fmt.Println("18. Regional Performance")
    fmt.Println("19. Course Competitiveness")
    fmt.Println("20. Exit")
    fmt.Print("\nEnter your choice (1-20): ")
}

func searchCandidates(db *sql.DB) {
    var searchTerm string
    fmt.Print("Enter registration number or surname to search: ")
    scanner := bufio.NewScanner(os.Stdin)
    if scanner.Scan() {
        searchTerm = scanner.Text()
    }

    query := `
        SELECT regnumber, surname, firstname, gender, aggregate 
        FROM candidates 
        WHERE regnumber LIKE $1 OR LOWER(surname) LIKE LOWER($1)
        LIMIT 10
    `

    rows, err := db.Query(query, "%"+searchTerm+"%")
    if err != nil {
        log.Printf("Error searching candidates: %v", err)
        return
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
}

func displayTopPerformers(db *sql.DB) {
    query := `
        SELECT regnumber, surname, firstname, aggregate 
        FROM candidates 
        WHERE aggregate IS NOT NULL 
        ORDER BY aggregate DESC 
        LIMIT 10
    `

    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting top performers: %v", err)
        return
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
}

func displayGenderStats(db *sql.DB) {
    query := `
        SELECT gender, COUNT(*) as count 
        FROM candidates 
        WHERE gender IS NOT NULL 
        GROUP BY gender
    `

    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting gender stats: %v", err)
        return
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
}

func displayStateDistribution(db *sql.DB) {
    query := `
        SELECT s.st_name, COUNT(c.*) as count 
        FROM candidates c
        JOIN state s ON c.statecode = s.st_id
        GROUP BY s.st_name 
        ORDER BY count DESC
        LIMIT 10
    `

    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting state distribution: %v", err)
        return
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
}

func displaySubjectStats(db *sql.DB) {
    query := `
        SELECT s.su_name, AVG(CASE 
            WHEN c.subj1 = s.su_id THEN c.score1
            WHEN c.subj2 = s.su_id THEN c.score2
            WHEN c.subj3 = s.su_id THEN c.score3
            WHEN c.subj4 = s.su_id THEN c.score4
        END) as avg_score
        FROM subject s
        JOIN candidates c ON 
            s.su_id IN (c.subj1, c.subj2, c.subj3, c.subj4)
        GROUP BY s.su_name
        ORDER BY avg_score DESC
    `

    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting subject stats: %v", err)
        return
    }
    defer rows.Close()

    color.Yellow("\nAverage Scores by Subject")
    table := tablewriter.NewWriter(os.Stdout)
    table.SetHeader([]string{"Subject", "Average Score"})

    for rows.Next() {
        var subject string
        var avgScore float64

        err := rows.Scan(&subject, &avgScore)
        if err != nil {
            continue
        }

        table.Append([]string{
            subject,
            fmt.Sprintf("%.2f", avgScore),
        })
    }

    table.Render()
}

func displayAggregateDistribution(db *sql.DB) {
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
        FROM candidates
        WHERE aggregate IS NOT NULL
        GROUP BY range
        ORDER BY range DESC
    `

    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting aggregate distribution: %v", err)
        return
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
}

func displayCourseAnalysis(db *sql.DB) {
    query := `
        SELECT c."COURSE NAME", COUNT(ca.regnumber) as applicants,
               ROUND(AVG(ca.aggregate)::numeric, 2) as avg_score,
               f.fac_name as faculty
        FROM courses c
        LEFT JOIN candidates ca ON c.corcode = ca.app_course1
        LEFT JOIN faculty f ON c.facid = f.fac_id
        GROUP BY c."COURSE NAME", f.fac_name
        ORDER BY applicants DESC
        LIMIT 15
    `
    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting course analysis: %v", err)
        return
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
}

func displayInstitutionStats(db *sql.DB) {
    query := `
        SELECT i.inname, COUNT(c.regnumber) as applicants,
               ROUND(AVG(c.aggregate)::numeric, 2) as avg_score,
               it.intyp_desc as institution_type
        FROM institutions i
        LEFT JOIN candidates c ON i.inid = c.inid
        LEFT JOIN institution_type it ON i.intyp = it.intyp_id
        GROUP BY i.inname, it.intyp_desc
        ORDER BY applicants DESC
        LIMIT 15
    `
    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting institution stats: %v", err)
        return
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
}

func displayFacultyPerformance(db *sql.DB) {
    query := `
        SELECT f.fac_name, COUNT(c.regnumber) as applicants,
               ROUND(AVG(c.aggregate)::numeric, 2) as avg_score
        FROM faculty f
        JOIN courses co ON f.fac_id = co.facid
        LEFT JOIN candidates c ON co.corcode = c.app_course1
        GROUP BY f.fac_name
        ORDER BY avg_score DESC
    `
    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting faculty performance: %v", err)
        return
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
}

func displayGeographicAnalysis(db *sql.DB) {
    query := `
        SELECT s.st_name as state, l.lg_name as lga,
               COUNT(c.regnumber) as candidates,
               ROUND(AVG(c.aggregate)::numeric, 2) as avg_score
        FROM state s
        JOIN lga l ON s.st_id = l.lg_st_id
        JOIN candidates c ON l.lg_id = c.lg_id
        GROUP BY s.st_name, l.lg_name
        HAVING COUNT(c.regnumber) > 1000
        ORDER BY candidates DESC
        LIMIT 15
    `
    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting geographic analysis: %v", err)
        return
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
}

func displayYearComparison(db *sql.DB) {
    query := `
        SELECT year,
               COUNT(*) as total_candidates,
               ROUND(AVG(aggregate)::numeric, 2) as avg_score,
               COUNT(CASE WHEN gender = 'F' THEN 1 END) as female_candidates,
               COUNT(CASE WHEN gender = 'M' THEN 1 END) as male_candidates
        FROM candidates
        GROUP BY year
        ORDER BY year
    `
    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting year comparison: %v", err)
        return
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
}

func displayAdmissionTrends(db *sql.DB) {
    query := `
        WITH course_stats AS (
            SELECT c."COURSE NAME",
                   COUNT(*) as applicants,
                   PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY ca.aggregate) as cutoff_score
            FROM courses c
            JOIN candidates ca ON c.corcode = ca.app_course1
            GROUP BY c."COURSE NAME"
            HAVING COUNT(*) > 100
        )
        SELECT "COURSE NAME",
               applicants,
               ROUND(cutoff_score::numeric, 2) as cutoff_score
        FROM course_stats
        ORDER BY applicants DESC
        LIMIT 15
    `
    rows, err := db.Query(query)
    if err != nil {
        log.Printf("Error getting admission trends: %v", err)
        return
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

func getNullableInt64(i sql.NullInt64) int64 {
    if i.Valid {
        return i.Int64
    }
    return 0
}

func handleCandidateImport(db *sql.DB) {
    fmt.Print("Enter the CSV file path: ")
    filename := readString()

    fmt.Print("Enter the year for the data (e.g., 2023): ")
    year := readInt()

    fmt.Print("Is this admission data? (y/n): ")
    isAdmission := strings.ToLower(readString()) == "y"

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
        // Open the CSV file
        file, err := os.Open(filename)
        if err != nil {
            color.Red("Error opening file: %v", err)
            return
        }
        defer file.Close()

        // Create CSV reader
        reader := csv.NewReader(file)

        config := importer.ImportConfig{
            Year:        year,
            SourceFile:  filename,
            IsAdmission: isAdmission,
            BatchSize:   1000,
            WorkerCount: workerCount,
        }

        if err := importer.ImportData(db, config, reader); err != nil {
            color.Red("Error importing data: %v", err)
        } else {
            color.Green("Import completed successfully!")
        }
    } else {
        fmt.Println("Import cancelled.")
    }
}

func handleAnalyzeFailedImports(db *sql.DB) {
    fmt.Print("Enter the path to the CSV file to analyze: ")
    filename := readString()

    workerCount := 4 // default value
    if envWorkerCount := os.Getenv("WORKER_COUNT"); envWorkerCount != "" {
        if count, err := strconv.Atoi(envWorkerCount); err == nil && count > 0 {
            workerCount = count
        }
    }

    fmt.Printf("\nUsing %d workers for parallel processing\n", workerCount)

    config := importer.ImportConfig{
        SourceFile:  filename,
        WorkerCount: workerCount,
    }
    imp := importer.NewDataImporter(db, config)

    _, err := imp.AnalyzeFailedImports(filename)
    if err != nil {
        color.Red("Error analyzing imports: %v", err)
        return
    }
}

func displayPerformanceMetrics(db *sql.DB) {
    query := `
        WITH ScoreStats AS (
            SELECT 
                year,
                COUNT(*) as total_candidates,
                AVG(NULLIF(aggregate, 0)) as avg_score,
                PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY NULLIF(aggregate, 0)) as median_score,
                STDDEV(NULLIF(aggregate, 0)) as std_dev
            FROM candidates 
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
    
    rows, err := db.Query(query)
    if err != nil {
        color.Red("Error fetching performance metrics: %v", err)
        return
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
}

func displayInstitutionRanking(db *sql.DB) {
    query := `
        WITH AdmissionStats AS (
            SELECT 
                i.inname as institution_name,
                i.inabv as abbreviation,
                COUNT(c.regnumber) as total_applicants,
                COUNT(CASE WHEN c.is_admitted = true THEN 1 END) as admitted_count,
                AVG(NULLIF(c.aggregate, 0)) as avg_score
            FROM institutions i
            LEFT JOIN candidates c ON i.inid = c.inid
            WHERE c.year = (SELECT MAX(year) FROM candidates)
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
        WHERE avg_score > 0
        ORDER BY avg_score DESC
        LIMIT 20;
    `
    
    rows, err := db.Query(query)
    if err != nil {
        color.Red("Error fetching institution rankings: %v", err)
        return
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
}

func displaySubjectCorrelation(db *sql.DB) {
    query := `
        WITH SubjectScores AS (
            SELECT 
                s1.su_name as english,
                s2.su_name as subject2,
                s3.su_name as subject3,
                s4.su_name as subject4,
                c.score1 as english_score,
                c.score2 as score2,
                c.score3 as score3,
                c.score4 as score4,
                c.aggregate as total_score
            FROM candidates c
            JOIN subject s1 ON c.subj1 = s1.su_id
            JOIN subject s2 ON c.subj2 = s2.su_id
            JOIN subject s3 ON c.subj3 = s3.su_id
            JOIN subject s4 ON c.subj4 = s4.su_id
            WHERE c.is_direct_entry IS NOT TRUE
            AND c.year = (SELECT MAX(year) FROM candidates)
            AND c.score2 > 0 AND c.score3 > 0 AND c.score4 > 0
        ),
        SubjectStats AS (
            SELECT 
                subject2 as subject1,
                subject3 as subject2,
                CORR(score2, score3) as correlation,
                COUNT(*) as sample_size,
                AVG(score2) as avg_score1,
                AVG(score3) as avg_score2,
                STDDEV(score2) as stddev1,
                STDDEV(score3) as stddev2,
                AVG(total_score) as avg_total
            FROM SubjectScores
            GROUP BY subject2, subject3
            HAVING COUNT(*) >= 100
            
            UNION ALL
            
            SELECT 
                subject2,
                subject4,
                CORR(score2, score4),
                COUNT(*),
                AVG(score2),
                AVG(score4),
                STDDEV(score2),
                STDDEV(score4),
                AVG(total_score)
            FROM SubjectScores
            GROUP BY subject2, subject4
            HAVING COUNT(*) >= 100
            
            UNION ALL
            
            SELECT 
                subject3,
                subject4,
                CORR(score3, score4),
                COUNT(*),
                AVG(score3),
                AVG(score4),
                STDDEV(score3),
                STDDEV(score4),
                AVG(total_score)
            FROM SubjectScores
            GROUP BY subject3, subject4
            HAVING COUNT(*) >= 100
        )
        SELECT 
            subject1 as "Subject 1",
            subject2 as "Subject 2",
            ROUND(correlation::numeric, 3) as correlation,
            sample_size,
            ROUND(avg_score1::numeric, 2) as avg_score1,
            ROUND(avg_score2::numeric, 2) as avg_score2,
            ROUND(stddev1::numeric, 2) as stddev1,
            ROUND(stddev2::numeric, 2) as stddev2,
            ROUND(avg_total::numeric, 2) as avg_total_score
        FROM SubjectStats
        WHERE correlation IS NOT NULL
        ORDER BY correlation DESC
        LIMIT 15;
    `
    
    rows, err := db.Query(query)
    if err != nil {
        color.Red("Error fetching subject correlations: %v", err)
        return
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
        "Avg Total",
    })

    hasData := false
    for rows.Next() {
        var subj1, subj2 string
        var correlation float64
        var sampleSize int
        var avgScore1, avgScore2, stdDev1, stdDev2, avgTotal float64
        
        if err := rows.Scan(&subj1, &subj2, &correlation, &sampleSize, &avgScore1, &avgScore2, &stdDev1, &stdDev2, &avgTotal); err != nil {
            color.Red("Error scanning row: %v", err)
            continue
        }
        
        hasData = true
        table.Append([]string{
            subj1,
            subj2,
            fmt.Sprintf("%.3f", correlation),
            fmt.Sprintf("%d", sampleSize),
            fmt.Sprintf("%.2f", avgScore1),
            fmt.Sprintf("%.2f", avgScore2),
            fmt.Sprintf("%.2f", stdDev1),
            fmt.Sprintf("%.2f", stdDev2),
            fmt.Sprintf("%.2f", avgTotal),
        })
    }

    color.Cyan("\nSubject Score Correlations (Latest Year)")
    if hasData {
        table.Render()
        color.Yellow("\nAnalysis of how subject scores relate to each other:")
        color.Yellow("1. Correlation shows how scores in two subjects move together")
        color.Yellow("2. Higher correlation (closer to +1.0) means students who do well in one subject")
        color.Yellow("   tend to do well in the other subject too")
        color.Yellow("3. Lower correlation (closer to -1.0) means opposite performance")
    } else {
        color.Yellow("\nNo significant correlations found between subjects.")
    }
}

func displayRegionalPerformance(db *sql.DB) {
    query := `
        WITH RegionalStats AS (
            SELECT 
                s.st_name as state_name,
                COUNT(c.regnumber) as total_candidates,
                AVG(NULLIF(c.aggregate, 0)) as avg_score,
                COUNT(CASE WHEN c.is_admitted = true THEN 1 END) as admitted_count,
                COUNT(CASE WHEN c.gender = 'F' THEN 1 END) as female_count
            FROM candidates c
            JOIN state s ON c.statecode = s.st_id
            WHERE c.year = (SELECT MAX(year) FROM candidates)
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
        ORDER BY average_score DESC;
    `
    
    rows, err := db.Query(query)
    if err != nil {
        color.Red("Error fetching regional performance: %v", err)
        return
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
}

func displayCourseCompetitiveness(db *sql.DB) {
    query := `
        WITH CourseStats AS (
            SELECT 
                c.app_course1 as course_code,
                co."COURSE NAME" as course_name,
                COUNT(c.regnumber) as total_applicants,
                MIN(NULLIF(c.aggregate, 0)) as min_score,
                MAX(NULLIF(c.aggregate, 0)) as max_score,
                AVG(NULLIF(c.aggregate, 0)) as avg_score,
                COUNT(CASE WHEN c.is_admitted = true THEN 1 END) as admitted_count
            FROM candidates c
            JOIN courses co ON c.app_course1 = co.corcode
            WHERE c.year = (SELECT MAX(year) FROM candidates)
                AND c.aggregate IS NOT NULL 
                AND c.aggregate > 0
            GROUP BY c.app_course1, co."COURSE NAME"
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
        WHERE avg_score > 0
        ORDER BY avg_score DESC
        LIMIT 20;
    `
    
    rows, err := db.Query(query)
    if err != nil {
        color.Red("Error fetching course competitiveness: %v", err)
        return
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
}

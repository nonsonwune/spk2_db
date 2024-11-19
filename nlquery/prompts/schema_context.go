package prompts

const SchemaContext = `Database Schema:

Tables:
1. candidate
   - regnumber (PK): Candidate registration number
   - year: Application year (e.g., 2020, 2021, 2022)
   - statecode: State code (FK to state.st_id)
   - app_course1: First choice course code (FK to course.course_code)
   - is_admitted: Boolean indicating if candidate was admitted (true/false)
   - inid: Institution ID (FK to institution.inid)

2. institution
   - inid (PK): Institution ID
   - inname: Institution name (e.g., "University of Lagos")
   - instate: State where institution is located
   - intype: Type of institution

3. state
   - st_id (PK): State ID
   - st_name: State name (e.g., "LAGOS")
   - st_code: State code

4. course
   - course_code (PK): Course code
   - course_name: Course name
   - faculty: Faculty name

Common Queries:
1. Top Institutions by Admissions:
   SELECT i.inname, COUNT(*) as total_admitted
   FROM candidate c
   JOIN institution i ON c.inid = i.inid
   WHERE c.is_admitted = true AND c.year = 2020
   GROUP BY i.inname
   ORDER BY total_admitted DESC
   LIMIT 5;

2. Course Statistics:
   SELECT c.course_name, COUNT(*) as total_applications
   FROM candidate ca
   JOIN course c ON ca.app_course1 = c.course_code
   WHERE ca.year = 2020
   GROUP BY c.course_name
   ORDER BY total_applications DESC;

3. State Distribution:
   SELECT s.st_name, COUNT(*) as total_candidates
   FROM candidate c
   JOIN state s ON c.statecode = s.st_id
   WHERE c.year = 2020
   GROUP BY s.st_name
   ORDER BY total_candidates DESC;`

const ExamplePatterns = `Query Pattern Examples:

1. Medicine-related Searches:
   - For named courses only:
     LOWER(course_name) LIKE ANY(ARRAY['%medicine%', '%medical%', '%surgery%', '%health%'])
     AND LOWER(course_name) NOT LIKE 'course%'
   
   - For all courses (including code-based):
     LOWER(course_name) LIKE ANY(ARRAY['%medicine%', '%medical%', '%surgery%', '%health%'])

2. Course Code Searches:
   - Exact match: course_code = '112838K'
   - Pattern match: course_code LIKE '1128%'

3. Named Courses Only:
   - Pattern: LOWER(course_name) NOT LIKE 'course%'
   
4. Code-Based Courses Only:
   - Pattern: LOWER(course_name) LIKE 'course%'

5. Engineering Courses:
   - Pattern: LOWER(course_name) LIKE '%engineering%'

6. Admission Statistics:
   - Basic: c.is_admitted = true
   - With year: c.is_admitted = true AND c.year = 2020
   - By institution: GROUP BY i.inname ORDER BY COUNT(*) DESC
   - By state: GROUP BY s.st_name ORDER BY COUNT(*) DESC

const QueryExamples = `Example Queries and Their SQL:

1. "Show me the top 5 institutions by number of admissions in 2020"
SQL:
SELECT i.inname as institution_name,
       COUNT(DISTINCT c.regnumber) as total_admitted
FROM candidate c
JOIN institution i ON c.inid = i.inid
WHERE c.is_admitted = true
AND c.year = 2020
GROUP BY i.inid, i.inname
ORDER BY total_admitted DESC
LIMIT 5;

2. "What state had the most medicine applicants?"
SQL:
SELECT s.st_name, COUNT(DISTINCT c.regnumber) as applicant_count
FROM candidate c
JOIN state s ON c.statecode = s.st_id
JOIN course co ON c.app_course1 = co.course_code
WHERE LOWER(co.course_name) LIKE '%medicine%'
AND c.year = 2023
GROUP BY s.st_name
ORDER BY applicant_count DESC
LIMIT 1;

3. "How many students applied for biochemistry from each state?"
SQL:
SELECT 
    s.st_name,
    COUNT(DISTINCT c.regnumber) as total_applicants,
    COUNT(DISTINCT CASE WHEN c.is_admitted = true THEN c.regnumber END) as admitted_count,
    ROUND(100.0 * COUNT(DISTINCT CASE WHEN c.is_admitted = true THEN c.regnumber END) / 
        NULLIF(COUNT(DISTINCT c.regnumber), 0), 2) as admission_rate
FROM candidate c
JOIN state s ON c.statecode = s.st_id
JOIN course co ON c.app_course1 = co.course_code
WHERE LOWER(co.course_name) LIKE '%biochemistry%'
AND c.year = 2023
GROUP BY s.st_name
ORDER BY total_applicants DESC;

4. "Which region has the highest number of engineering applicants?"
SQL:
WITH RegionalStats AS (
    SELECT 
        CASE 
            WHEN s.st_name IN ('BENUE', 'FCT', 'KOGI', 'KWARA', 'NASARAWA', 'NIGER', 'PLATEAU') THEN 'North Central'
            WHEN s.st_name IN ('ADAMAWA', 'BAUCHI', 'BORNO', 'GOMBE', 'TARABA', 'YOBE') THEN 'North East'
            WHEN s.st_name IN ('JIGAWA', 'KADUNA', 'KANO', 'KATSINA', 'KEBBI', 'SOKOTO', 'ZAMFARA') THEN 'North West'
            WHEN s.st_name IN ('ABIA', 'ANAMBRA', 'EBONYI', 'ENUGU', 'IMO') THEN 'South East'
            WHEN s.st_name IN ('AKWA IBOM', 'BAYELSA', 'CROSS RIVER', 'DELTA', 'EDO', 'RIVERS') THEN 'South South'
            WHEN s.st_name IN ('EKITI', 'LAGOS', 'OGUN', 'ONDO', 'OSUN', 'OYO') THEN 'South West'
        END as region,
        COUNT(DISTINCT c.regnumber) as total_applicants
    FROM candidate c
    JOIN state s ON c.statecode = s.st_id
    JOIN course co ON c.app_course1 = co.course_code
    WHERE LOWER(co.course_name) LIKE '%engineering%'
    AND c.year = 2023
    GROUP BY 
        CASE 
            WHEN s.st_name IN ('BENUE', 'FCT', 'KOGI', 'KWARA', 'NASARAWA', 'NIGER', 'PLATEAU') THEN 'North Central'
            WHEN s.st_name IN ('ADAMAWA', 'BAUCHI', 'BORNO', 'GOMBE', 'TARABA', 'YOBE') THEN 'North East'
            WHEN s.st_name IN ('JIGAWA', 'KADUNA', 'KANO', 'KATSINA', 'KEBBI', 'SOKOTO', 'ZAMFARA') THEN 'North West'
            WHEN s.st_name IN ('ABIA', 'ANAMBRA', 'EBONYI', 'ENUGU', 'IMO') THEN 'South East'
            WHEN s.st_name IN ('AKWA IBOM', 'BAYELSA', 'CROSS RIVER', 'DELTA', 'EDO', 'RIVERS') THEN 'South South'
            WHEN s.st_name IN ('EKITI', 'LAGOS', 'OGUN', 'ONDO', 'OSUN', 'OYO') THEN 'South West'
        END
)
SELECT region, total_applicants
FROM RegionalStats
ORDER BY total_applicants DESC
LIMIT 1;`

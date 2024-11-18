package prompts

const SchemaContext = `Database Schema and Relationships:

Tables and Their Relationships:
1. candidate
   - Primary Key: regnumber
   - Foreign Keys:
     * statecode -> state.st_id (candidate's state)
     * inid -> institution.inid (institution applied to)
     * lg_id -> lga.lg_id (local government area)
     * app_course1 -> course.course_code (applied course)

2. institution
   - Primary Key: inid
   - Foreign Keys:
     * inst_state_id -> state.st_id (institution's state)
     * affiliated_state_id -> state.st_id (affiliated state)
     * intyp -> institution_type.intyp_id (institution type)

3. course
   - Primary Key: course_code
   - Foreign Keys:
     * facid -> faculty.fac_id (faculty)

Common Query Patterns:
1. Candidate Statistics:
   - Count by gender, state, institution
   - Aggregate scores analysis
   - Admission status

2. Institution Analysis:
   - Admission trends
   - Course offerings
   - State distribution

3. Course Analysis:
   - Popular courses
   - Faculty distribution
   - Historical changes

Important Notes:
1. Always use COUNT(DISTINCT) for accurate counting
2. Consider temporal aspects (year field)
3. Handle NULL values appropriately
4. Use proper table aliases
5. Consider performance with large datasets`

const ExamplePatterns = `
- For medicine-related queries, include variations like 'MEDICINE', 'MEDICAL', 'SURGERY'
- When searching for courses, use ILIKE with wildcards (%) for flexible matching
- For counting applications, consider grouping by course_name to show distribution
- Example medicine query:
  SELECT COUNT(*), co.course_name 
  FROM candidate c 
  JOIN course co ON c.app_course1 = co.course_code 
  WHERE co.course_name ILIKE '%MEDICINE%' OR co.course_name ILIKE '%MEDICAL%'
  GROUP BY co.course_name;
`

const QueryExamples = `Example Queries and Their SQL:

1. "How many female candidates were admitted to medical courses in 2019?"
SQL:
SELECT COUNT(DISTINCT c.regnumber)
FROM candidate c
JOIN course co ON c.app_course1 = co.course_code
WHERE c.gender = 'F'
AND c.is_admitted = true
AND c.year = 2019
AND co.course_name ILIKE '%medicine%';

2. "Show me the top 5 institutions by number of admissions in 2020"
SQL:
SELECT i.inname, COUNT(DISTINCT c.regnumber) as admitted_count
FROM institution i
JOIN candidate c ON i.inid = c.inid
WHERE c.is_admitted = true
AND c.year = 2020
GROUP BY i.inid, i.inname
ORDER BY admitted_count DESC
LIMIT 5;`

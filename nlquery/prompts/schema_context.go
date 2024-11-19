package prompts

const SchemaContext = `Database Schema and Course Information:

1. Database Statistics:
   - Named courses: 1,474 (with descriptive names)
   - Code-only courses: 3,037 (format: course_name = "Course " + course_code)
   - Total courses: 4,511

2. Course Code Structure:
   - All courses have a unique course_code (e.g., "112838K", "12267C")
   - Some courses have descriptive names (e.g., "MEDICINE & SURGERY")
   - Others use code-based names where course_name = "Course " + course_code
   - IMPORTANT: Code-based courses are valid courses, not placeholders

3. Tables and Their Relationships:
   - faculty
     * Primary Key: fac_id (integer)
     * Columns:
       - fac_code: Faculty code (varchar(10))
       - fac_name: Full faculty name (varchar(100))
       - fac_abv: Faculty abbreviation (varchar(20))
     * Referenced by:
       - course.facid -> faculty.fac_id

   - subject
     * Primary Key: su_id (integer)
     * Columns:
       - su_abrv: Subject abbreviation (varchar(10))
       - su_name: Full subject name (varchar(100))
     * Referenced by:
       - candidate_scores.subject_id -> subject.su_id
       - subject_mapping_2023.su_id -> subject.su_id
     * Available Subjects:
       1. Use of English (Core)
       2. Mathematics (Core)
       3. Sciences:
          - Biology
          - Chemistry
          - Physics
          - Computer Studies
          - Agriculture
       4. Arts and Humanities:
          - Literature in English
          - History
          - Government
          - Economics
          - Geography
       5. Languages:
          - Arabic
          - French
          - Hausa
          - Igbo
          - Yoruba
       6. Other Subjects:
          - Art (Fine Art)
          - Commerce
          - Home Economics
          - Islamic Studies
          - Christian Religious Knowledge
          - Music
          - Physical and Health Education
          - Principles of Accounts

   - candidate
     * Primary Key: regnumber
     * Foreign Keys:
       - statecode -> state.st_id (candidate's state)
       - inid -> institution.inid (institution applied to)
       - lg_id -> lga.lg_id (local government area)
       - app_course1 -> course.course_code (applied course)

   - course
     * Primary Key: course_code
     * course_name can be either:
       - Descriptive name (e.g., "MEDICINE & SURGERY")
       - Code-based name (e.g., "Course 112838K")
     * Foreign Keys:
       - facid -> faculty.fac_id (faculty)

   - institution_type
     * Primary Key: intyp_id (integer)
     * Columns:
       - intyp_desc: Institution type description (varchar(100))
       - inst_cat: Institution category (varchar(20))
     * Referenced by:
       - institution.intyp -> institution_type.intyp_id

   - institution
     * Primary Key: inid
     * Foreign Keys:
       - inst_state_id -> state.st_id (institution's state)
       - affiliated_state_id -> state.st_id (affiliated state)
       - intyp -> institution_type.intyp_id (institution type)

4. Course Categories (Named Courses):
   A. Medicine and Health Sciences:
      - Medicine & Surgery
      - Medical Laboratory Science
      - Optometry
      - Pharmacy
      - Public Health
      - Veterinary Medicine

   B. Engineering:
      - Aerospace Engineering
      - Biomedical Engineering
      - Chemical Engineering
      - Civil Engineering
      - Computer Engineering
      - Electrical Engineering
      - Mechanical Engineering

   C. Sciences:
      - Biochemistry
      - Biology
      - Chemistry
      - Computer Science
      - Mathematics
      - Physics
      - Statistics

   D. Social Sciences:
      - Economics
      - Geography
      - Political Science
      - Psychology
      - Sociology

   E. Arts and Humanities:
      - English Language
      - History
      - Islamic Studies
      - Languages (Arabic, French, Hausa, Yoruba)
      - Religious Studies

   F. Education:
      - Adult Education
      - Guidance & Counselling
      - Science Education
      - Special Education

   G. Business and Management:
      - Accounting
      - Business Administration
      - Marketing
      - Project Management

   H. Agriculture:
      - Agricultural Economics
      - Agricultural Engineering
      - Animal Science
      - Crop Science
      - Fisheries

5. Query Best Practices:
   - Always use COUNT(DISTINCT) for accurate counting
   - Consider temporal aspects (year field)
   - Handle NULL values appropriately
   - Use proper table aliases
   - Consider performance with large datasets
   - When searching for specific courses:
     * Include both named and code-based courses unless specifically filtering
     * Use course_code for exact matches
     * Use LIKE on course_name for pattern matching`

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
   - Include technology variants: '%engineering technology%'

6. Science Programs:
   - Include both pure and applied:
     LOWER(course_name) LIKE ANY(ARRAY['%science%', '%biology%', '%chemistry%', '%physics%'])

7. Language Courses:
   - Include specific languages:
     LOWER(course_name) LIKE ANY(ARRAY['%english%', '%french%', '%arabic%', '%hausa%', '%yoruba%'])

8. Business Studies:
   - Pattern: LOWER(course_name) LIKE ANY(ARRAY['%business%', '%management%', '%accounting%'])

9. Education Programs:
   - Include variations:
     LOWER(course_name) LIKE ANY(ARRAY['%education%', '%teaching%', '%pedagogy%'])`

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

package prompts

const SchemaContext = `Database Schema:

Tables:
1. candidate
   - regnumber (PK): varchar(20), Candidate registration number
   - maritalstatus: varchar(20), Marital status
   - is_blind: boolean, Indicates if candidate is blind
   - is_deaf: boolean, Indicates if candidate is deaf
   - address: text, Candidate's address
   - noofsittings: integer, Number of JAMB sittings
   - is_mock_candidate: boolean, If candidate took mock exam
   - email: varchar(100), Email address
   - gsmno: varchar(20), Phone number
   - surname: varchar(100), Last name
   - firstname: varchar(100), First name
   - middlename: varchar(100), Middle name
   - gender: varchar(10), Gender (Male/Female)
   - statecode: integer (FK -> state.st_id), State of origin
   - aggregate: integer, Total aggregate score
   - app_course1: varchar(100) (FK -> course.course_code), First choice course
   - inid: varchar(20) (FK -> institution.inid), Institution ID
   - lg_id: integer (FK -> lga.lg_id), Local government area
   - year: integer (NOT NULL), Application year
   - is_admitted: boolean, Admission status
   - is_direct_entry: boolean, Direct entry status
   - malpractice: text, Any recorded malpractice
   - date_of_birth: date, Date of birth
   - saved_at: timestamp, Record save timestamp
   - created_at: timestamp, Record creation timestamp
   - updated_at: timestamp, Record last update timestamp

2. candidate_scores
   - cand_reg_number (PK, FK): varchar(20), References candidate.regnumber
   - subject_id (PK, FK): integer, References subject.su_id
   - score: integer, Score in the subject
   - year (PK): integer, Year of exam

3. subject
   - su_id (PK): integer, Subject ID
   - su_abrv: varchar(10), Subject abbreviation
   - su_name: varchar(100), Subject name (e.g., Mathematics, English)

4. state
   - st_id (PK): integer, State ID
   - st_name: varchar(100), State name (ALL CAPS, e.g., "LAGOS", "ANAMBRA")
   - st_code: varchar(10), State code

5. course
   - course_code (PK): varchar(100), Course code
   - instid: integer, Institution ID
   - facid: integer (FK -> faculty.fac_id), Faculty ID
   - group_id: integer, Course group ID
   - course_name: varchar(200), Course name
   - course_abbreviation: varchar(50), Course abbreviation
   - duration: integer, Course duration in years
   - degree: varchar(50), Degree type

6. faculty
   - fac_id (PK): integer, Faculty ID
   - fac_name: varchar(100), Faculty name
   - fac_code: varchar(10), Faculty code

7. institution
   - inid (PK): varchar(20), Institution ID
   - inname: varchar(200), Institution name
   - instate: integer (FK -> state.st_id), State where institution is located
   - intype: varchar(50), Institution type

8. analysis_cache
   - id (PK): integer, Cache entry ID
   - query_hash: text, Hash of the query for caching
   - query_text: text, Original query text
   - analysis_result: jsonb, Cached analysis result
   - created_at: timestamp, Cache entry creation time

9. candidate_disabilities
   - cand_reg_number (PK, FK): varchar(20), References candidate.regnumber
   - is_blind: boolean, Visual impairment status
   - is_deaf: boolean, Hearing impairment status
   - other_challenges: varchar(100), Other disability descriptions

10. candidate_exam_info
    - cand_reg_number (PK, FK): varchar(20), References candidate.regnumber
    - exam_town: varchar(100), Town where exam was taken
    - exam_centre: varchar(200), Examination center
    - exam_number: varchar(20), Candidate's exam number
    - mock_state_id (FK): integer, References state.st_id for mock exam
    - mock_town: varchar(100), Town for mock exam
    - is_mock_candidate: boolean, Mock exam participation status

11. course_code_mappings
    - id (PK): integer, Mapping entry ID
    - year: integer, Academic year
    - old_course_code: varchar(100), Previous course code
    - new_course_code: varchar(100), Updated course code
    - course_name: varchar(200), Course name
    - institution_id: integer, Institution identifier
    - mapping_date: timestamp, When mapping was created

Common Query Patterns:

1. Subject Scores:
   SELECT cs.score, s.su_name
   FROM candidate_scores cs
   JOIN subject s ON cs.subject_id = s.su_id
   WHERE cs.cand_reg_number = '12345678' AND cs.year = 2023;

2. State-based Queries:
   - Use UPPER case for state names: s.st_name = 'LAGOS'
   - Join path: candidate.statecode -> state.st_id

3. Course Queries:
   - Exact match: UPPER(co.course_name) = 'MEDICINE'
   - Pattern match: LOWER(co.course_name) LIKE LOWER('%medicine%')
   - Multiple courses: UPPER(co.course_name) IN ('MEDICINE', 'SURGERY')

4. Gender Statistics:
   - Filter: c.gender = 'Female' OR c.gender = 'Male'
   - Group by: GROUP BY c.gender

5. Subject Analysis:
   - Join path: candidate_scores.subject_id -> subject.su_id
   - Subject names: Use exact match with subject.su_name
   - Scores: candidate_scores.score for individual scores

6. Admission Analysis:
   - Filter: c.is_admitted = true
   - Join with institution: candidate.inid -> institution.inid
   - Join with course: candidate.app_course1 -> course.course_code
`

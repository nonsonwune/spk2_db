-- Standardize table names to use singular form
ALTER TABLE institutions RENAME TO institution;
ALTER TABLE courses RENAME TO course;
ALTER TABLE candidates RENAME TO candidate;

-- Update foreign key constraints
ALTER TABLE candidate 
  DROP CONSTRAINT IF EXISTS fk_institution,
  ADD CONSTRAINT fk_institution 
  FOREIGN KEY (inid) 
  REFERENCES institution(inid);

ALTER TABLE institution_names 
  DROP CONSTRAINT IF EXISTS institution_names_inid_fkey,
  ADD CONSTRAINT institution_names_inid_fkey 
  FOREIGN KEY (inid) 
  REFERENCES institution(inid);

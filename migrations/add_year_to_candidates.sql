-- Rename candidates_2021 to candidates
ALTER TABLE candidates_2021 RENAME TO candidates;

-- Add year column to candidates table
ALTER TABLE candidates ADD COLUMN year integer;

-- Update existing records with year 2021
UPDATE candidates SET year = 2021;

-- Make year column NOT NULL
ALTER TABLE candidates ALTER COLUMN year SET NOT NULL;

-- Create an index on year column for better query performance
CREATE INDEX idx_candidates_year ON candidates(year);

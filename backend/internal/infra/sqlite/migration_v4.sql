ALTER TABLE user ADD COLUMN education_stage TEXT NOT NULL DEFAULT 'university' CHECK (education_stage IN ('university','highschool'));
ALTER TABLE custom_subject ADD COLUMN education_stage TEXT NOT NULL DEFAULT 'university' CHECK (education_stage IN ('university','highschool'));

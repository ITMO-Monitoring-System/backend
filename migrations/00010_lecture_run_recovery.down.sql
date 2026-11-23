DROP INDEX universities_data.lecture_runs_unfinished_idx;
ALTER TABLE universities_data.lecture_runs DROP COLUMN started_at, DROP COLUMN interrupted;

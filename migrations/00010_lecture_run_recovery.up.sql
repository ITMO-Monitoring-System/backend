ALTER TABLE universities_data.lecture_runs
    ADD COLUMN started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    ADD COLUMN interrupted boolean NOT NULL DEFAULT false;

CREATE INDEX lecture_runs_unfinished_idx
    ON universities_data.lecture_runs (lecture_id) WHERE NOT finished;

ALTER TABLE visits.lectures_visiting
    ADD COLUMN run_id UUID,
    ADD CONSTRAINT lectures_visiting_run_user_unique UNIQUE (run_id, user_id);

CREATE TABLE visits.lecture_observations (
    run_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (run_id, request_id),
    FOREIGN KEY (run_id, user_id)
        REFERENCES visits.lectures_visiting (run_id, user_id) ON DELETE CASCADE
);

CREATE INDEX lecture_observations_run_user_time_idx
    ON visits.lecture_observations (run_id, user_id, observed_at);

CREATE VIEW visits.lecture_observation_history AS
    SELECT v.lecture_id, v.user_id, v.run_id, o.observed_at
    FROM visits.lectures_visiting v
    JOIN visits.lecture_observations o USING (run_id, user_id)
    UNION ALL
    SELECT lecture_id, user_id, run_id, date
    FROM visits.lectures_visiting
    WHERE run_id IS NULL;

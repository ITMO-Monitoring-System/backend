DROP VIEW visits.lecture_observation_history;

-- Preserve the observation history when returning to the legacy schema.
UPDATE visits.lectures_visiting AS v
SET date = first_observation.observed_at
FROM (
    SELECT run_id, user_id, MIN(observed_at) AS observed_at
    FROM visits.lecture_observations GROUP BY run_id, user_id
) AS first_observation
WHERE (v.run_id, v.user_id) = (first_observation.run_id, first_observation.user_id);

INSERT INTO visits.lectures_visiting (lecture_id, user_id, date)
SELECT v.lecture_id, v.user_id, o.observed_at
FROM visits.lectures_visiting v
JOIN (
    SELECT *, ROW_NUMBER() OVER (PARTITION BY run_id, user_id ORDER BY observed_at, request_id) AS row_number
    FROM visits.lecture_observations
) o USING (run_id, user_id)
WHERE o.row_number > 1;

DROP TABLE visits.lecture_observations;
ALTER TABLE visits.lectures_visiting DROP COLUMN run_id;

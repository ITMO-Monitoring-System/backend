CREATE TABLE universities_data.lecture_runs (
    lecture_id bigint NOT NULL REFERENCES universities_data.lectures(id) ON DELETE CASCADE,
    run_id text NOT NULL,
    finished boolean NOT NULL DEFAULT false,
    PRIMARY KEY (lecture_id, run_id)
);

INSERT INTO universities_data.lecture_runs (lecture_id, run_id, finished)
SELECT DISTINCT lecture_id, COALESCE(run_id::text, ''), true FROM visits.lectures_visiting;

CREATE VIEW universities_data.lecture_catalog AS
SELECT l.id, l.date, l.subject_id, l.teacher_id, s.name AS subject_name,
    ARRAY(SELECT lg.group_id FROM universities_data.lectures_groups lg
          WHERE lg.lecture_id = l.id ORDER BY lg.group_id) AS group_ids,
    ARRAY(SELECT DISTINCT g.department_id FROM universities_data.lectures_groups lg
          JOIN universities_data.groups g ON g.code = lg.group_id
          WHERE lg.lecture_id = l.id ORDER BY g.department_id) AS department_ids,
    ARRAY(SELECT DISTINCT d.code FROM universities_data.lectures_groups lg
          JOIN universities_data.groups g ON g.code = lg.group_id
          JOIN universities_data.departments d ON d.id = g.department_id
          WHERE lg.lecture_id = l.id ORDER BY d.code) AS department_codes,
    CASE (SELECT bool_and(finished) FROM universities_data.lecture_runs r WHERE r.lecture_id = l.id)
        WHEN true THEN 'completed' WHEN false THEN 'running' ELSE 'scheduled'
    END AS status
FROM universities_data.lectures l JOIN universities_data.subjects s ON s.id = l.subject_id;

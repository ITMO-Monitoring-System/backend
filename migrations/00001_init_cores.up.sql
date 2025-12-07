create schema if not exists "cores";

create table if not exists cores.users (
    isu TEXT PRIMARY KEY,
    first_name VARCHAR(125) NOT NULL,
    last_name VARCHAR(125) NOT NULL,
    patronymic VARCHAR(125)
);

create table if not exists cores.face_images (
    student_id TEXT PRIMARY KEY,
    left_face bytea NOT NULL,
    right_face bytea NOT NULL,
    full_face bytea NOT NULL,
    FOREIGN KEY (student_id) REFERENCES cores.users(isu)
)
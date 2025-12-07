create schema if not exists "universities_data";

create table if not exists universities_data.departments(
    id BIGINT PRIMARY KEY,
    code VARCHAR(125) NOT NULL UNIQUE,
    name VARCHAR(125) NOT NULL UNIQUE,
    alias VARCHAR(125) UNIQUE
);

create table if not exists universities_data.groups (
    code VARCHAR(25) PRIMARY KEY,
    department_id BIGINT NOT NULL,
    FOREIGN KEY (department_id) REFERENCES universities_data.departments(id)
);

create table if not exists universities_data.students_groups
(
    user_id    TEXT PRIMARY KEY,
    group_code VARCHAR(25),
    foreign key (user_id) REFERENCES cores.users (isu),
    foreign key (group_code) REFERENCES universities_data.groups (code),
    UNIQUE (user_id, group_code)
);

CREATE INDEX IF NOT EXISTS idx_students_groups_group_code
    ON universities_data.students_groups(group_code);

create table if not exists universities_data.subjects(
    id bigint primary key,
    name VARCHAR(125) NOT NULL UNIQUE
);

create table if not exists universities_data.lectures (
    id BIGINT PRIMARY KEY,
    date timestamptz not null,
    subject_id BIGINT NOT NULL,
    teacher_id TEXT NOT NULL ,
    foreign key (subject_id) REFERENCES universities_data.subjects(id),
    foreign key (teacher_id) REFERENCES cores.users(isu)
);

create index if not exists idx_teacher_lectures
    ON universities_data.lectures(teacher_id);

create table if not exists universities_data.lectures_groups (
    id BIGINT PRIMARY KEY ,
    lecture_id BIGINT NOT NULL,
    group_id VARCHAR(25) NOT NULL,
    foreign key (lecture_id) references universities_data.lectures(id),
    foreign key (group_id) references universities_data.groups(code),
    UNIQUE (lecture_id, group_id)
);

create index if not exists idx_lectures_groups_lecture_id
    ON universities_data.lectures_groups(lecture_id);

create index if not exists idx_lectures_groups_group_id
    ON universities_data.lectures_groups(group_id);

create table if not exists universities_data.practices (
    id BIGINT PRIMARY KEY ,
    date timestamptz NOT NULL ,
    subject_id BIGINT NOT NULL ,
    teacher_id TEXT NOT NULL ,
    foreign key (subject_id) references universities_data.subjects(id),
    foreign key (teacher_id) references cores.users(isu)
);

create index if not exists idx_practices_teacher
    ON universities_data.practices(teacher_id);

create table if not exists universities_data.practices_groups (
    id BIGINT primary key ,
    practice_id BIGINT NOT NULL ,
    group_id VARCHAR(25) NOT NULL ,
    foreign key (practice_id) references universities_data.practices(id),
    foreign key (group_id) references universities_data.groups(code),
    UNIQUE (practice_id, group_id)
);

create index if not exists idx_practices_groups_practice_id
    ON universities_data.practices_groups(practice_id);

create index if not exists idx_practices_groups_group_id
    ON universities_data.practices_groups(group_id);


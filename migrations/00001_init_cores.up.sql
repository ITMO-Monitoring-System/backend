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
    left_face_embedding FLOAT4[] NOT NULL,
    right_face bytea NOT NULL,
    right_face_embedding float4[] NOT NULL,
    full_face bytea NOT NULL,
    full_face_embedding float4[] NOT NULL,
    FOREIGN KEY (student_id) REFERENCES cores.users(isu)
);

create table if not exists cores.users_passwords (
    isu text PRIMARY KEY,
    password TEXT NOT NULL,
    FOREIGN KEY (isu) REFERENCES cores.users(isu)
);

create table if not exists cores.users_roles (
    id serial primary key,
    isu text not null,
    role varchar(125) not null,
    unique (isu, role),
    foreign key (isu) references cores.users(isu)
);
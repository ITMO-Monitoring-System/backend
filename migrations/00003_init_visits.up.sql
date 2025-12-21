create schema if not exists "visits";

create table if not exists visits.lectures_visiting (
    id SERIAL PRIMARY KEY ,
    lecture_id BIGINT NOT NULL,
    user_id TEXT NOT NULL,
    date timestamptz not null,
    foreign key (lecture_id) references universities_data.lectures(id),
    foreign key (user_id) references cores.users(isu)
);

create index if not exists idx_lecture_visiting_lecture_id
    on visits.lectures_visiting(lecture_id);

create index if not exists idx_lecture_visiting_user_id
    on visits.lectures_visiting(user_id);

create index if not exists idx_lecture_visiting_user_id_date
    on visits.lectures_visiting(user_id, date);

create index if not exists idx_lecture_visiting_date
    on visits.lectures_visiting(date);

create index if not exists idx_lecture_visiting_lecture_id_date
    on visits.lectures_visiting(lecture_id, date);


create table if not exists visits.practices_visiting (
    id SERIAL PRIMARY KEY ,
    practice_id BIGINT not null ,
    user_id TEXT NOT NULL ,
    date timestamptz not null,
    foreign key (practice_id) references universities_data.practices(id),
    foreign key (user_id) references cores.users(isu)
);

create index if not exists idx_practices_visiting_practice_id
    on visits.practices_visiting(practice_id);

create index if not exists idx_practices_visiting_user_id
    on visits.practices_visiting(user_id);

create index if not exists idx_practices_visiting_user_id_date
    on visits.practices_visiting(user_id, date);

create index if not exists idx_practices_visiting_date
    on visits.practices_visiting(date);

create index if not exists idx_practices_visiting_practice_id_date
    on visits.practices_visiting(practice_id, date);
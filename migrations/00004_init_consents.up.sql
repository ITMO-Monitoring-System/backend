create table if not exists cores.user_consents (
    id serial primary key,
    isu text not null,
    consent_type text not null,
    doc_version text not null,
    accepted_at timestamptz not null default now(),
    revoked_at timestamptz,
    ip_address text,
    user_agent text,
    foreign key (isu) references cores.users(isu) on delete cascade
);

create index if not exists idx_user_consents_isu on cores.user_consents (isu);

-- +goose Up
create table mission_assignments (
    account_id uuid not null references accounts (id),
    mission_id text not null,
    period text not null,
    mission_type text not null,
    slot int not null,
    progress bigint not null default 0,
    assigned_at timestamptz not null,
    completed_at timestamptz,
    replaced_at timestamptz,
    primary key (account_id, mission_id, period)
);

create index mission_assignments_period on mission_assignments (account_id, period, slot);

create table mission_matches (
    account_id uuid not null references accounts (id),
    match_id text not null,
    created_at timestamptz not null,
    primary key (account_id, match_id)
);

-- +goose Down
drop table mission_matches;
drop table mission_assignments;

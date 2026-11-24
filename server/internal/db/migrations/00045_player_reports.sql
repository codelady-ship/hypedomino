-- +goose Up
create table player_reports (
    id bigint generated always as identity primary key,
    reporter uuid not null references accounts (id),
    target uuid not null references accounts (id),
    reason text not null check (reason in ('offensive_nickname', 'offensive_room_name', 'impersonation', 'other')),
    name_text text not null,
    match_id uuid,
    state text not null default 'open' check (state in ('open', 'upheld', 'rejected')),
    created_at timestamptz not null,
    updated_at timestamptz not null,
    decided_at timestamptz,
    decided_by text,
    check (reporter <> target)
);

create unique index player_reports_open on player_reports (reporter, target) where state = 'open';
create index player_reports_target on player_reports (target, state, reason);
create index player_reports_reporter on player_reports (reporter, updated_at);

-- +goose Down
drop table player_reports;

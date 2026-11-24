-- +goose Up
create table pass_xp_events (
    account_id uuid not null references accounts (id),
    season_id text not null,
    amount integer not null check (amount > 0),
    source_type text not null,
    source_ref text not null,
    idempotency_key text not null,
    match_xp boolean not null,
    economy_day date not null,
    created_at timestamptz not null,
    primary key (account_id, idempotency_key)
);

create index pass_xp_events_match_day on pass_xp_events (account_id, economy_day) where match_xp;

-- +goose Down
drop table pass_xp_events;

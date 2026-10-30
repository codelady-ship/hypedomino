-- +goose Up
create table rating_signals (
    account_id uuid not null references accounts (id),
    kind text not null,
    counterpart uuid not null references accounts (id),
    mode text not null check (mode in ('1v1', '2v2')),
    matches integer not null,
    wins integer not null,
    offenses integer not null,
    window_hours integer not null,
    first_seen timestamptz not null,
    updated_at timestamptz not null,
    primary key (account_id, kind, counterpart, mode)
);
create index rating_signals_updated_idx on rating_signals (updated_at);

-- +goose Down
drop table rating_signals;

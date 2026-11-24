-- +goose Up
create table risk_signals (
    id bigserial primary key,
    account_id uuid not null references accounts (id),
    kind text not null,
    weight integer not null,
    dedupe_key text not null,
    evidence jsonb not null,
    created_at timestamptz not null,
    unique (account_id, kind, dedupe_key)
);

create index risk_signals_account_idx on risk_signals (account_id, created_at desc);

-- +goose Down
drop table risk_signals;

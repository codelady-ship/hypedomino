-- +goose Up
create table reward_grants (
    id bigint generated always as identity primary key,
    account_id uuid not null references accounts (id),
    bundle_id text not null,
    revision integer not null check (revision >= 1),
    source_type text not null,
    source_ref text not null,
    idempotency_key text not null,
    config_version integer not null,
    report jsonb not null,
    created_at timestamptz not null default now(),
    revoked_at timestamptz,
    revoke_key text,
    unique (account_id, idempotency_key)
);
create index reward_grants_account_idx on reward_grants (account_id, created_at desc);

-- +goose Down
drop table reward_grants;

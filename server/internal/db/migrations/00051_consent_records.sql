-- +goose Up
create table consent_records (
    id bigint generated always as identity primary key,
    account_id uuid not null references accounts (id),
    purpose text not null check (purpose in ('analytics', 'diagnostics', 'promotional_push')),
    granted boolean not null,
    policy_version text not null,
    source text not null check (source in ('first_launch', 'settings')),
    app_version text not null,
    recorded_at timestamptz not null
);
create index consent_records_account_idx on consent_records (account_id, recorded_at);

-- +goose Down
drop table consent_records;

-- +goose Up
create table admin_sessions (
    id uuid primary key default gen_random_uuid(),
    account_id uuid not null references admin_accounts (id),
    token_hash bytea not null,
    created_at timestamptz not null,
    authenticated_at timestamptz not null,
    last_seen_at timestamptz not null,
    expires_at timestamptz not null,
    revoked_at timestamptz
);
create unique index admin_sessions_token_idx on admin_sessions (token_hash);
create index admin_sessions_account_idx on admin_sessions (account_id) where revoked_at is null;

-- +goose Down
drop table admin_sessions;

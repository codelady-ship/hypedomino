-- +goose Up
create table data_exports (
    id uuid primary key default gen_random_uuid(),
    account_id uuid not null references accounts (id),
    object_key text not null unique,
    requested_at timestamptz not null,
    expires_at timestamptz not null,
    file_deleted_at timestamptz
);
create index data_exports_account_idx on data_exports (account_id, requested_at desc);
create index data_exports_expiry_idx on data_exports (expires_at) where file_deleted_at is null;

-- +goose Down
drop table data_exports;

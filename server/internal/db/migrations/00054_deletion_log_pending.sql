-- +goose Up
create table deletion_log_pending (
    account_id uuid primary key,
    tombstone_id uuid not null,
    deleted_at timestamptz not null
);

-- +goose Down
drop table deletion_log_pending;

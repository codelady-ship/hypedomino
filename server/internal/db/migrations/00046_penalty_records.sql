-- +goose Up
create table penalty_records (
    account_id uuid primary key references accounts (id),
    record jsonb not null,
    updated_at timestamptz not null
);

-- +goose Down
drop table penalty_records;

-- +goose Up
create table emblem_showcases (
    account_id uuid primary key references accounts (id),
    emblems text[] not null,
    updated_at timestamptz not null
);

-- +goose Down
drop table emblem_showcases;

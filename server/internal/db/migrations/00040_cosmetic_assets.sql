-- +goose Up
create table cosmetic_assets (
    id uuid primary key,
    item_id text not null,
    slot text not null,
    address text not null,
    object_key text not null unique,
    status text not null check (status in ('awaiting_file', 'passed', 'rejected')),
    format text,
    width integer,
    height integer,
    bytes bigint,
    frames integer,
    content_hash text,
    checks jsonb,
    uploaded_by text not null,
    created_at timestamptz not null,
    checked_at timestamptz
);

-- +goose Down
drop table cosmetic_assets;

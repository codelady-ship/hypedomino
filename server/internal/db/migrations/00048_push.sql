-- +goose Up
create table push_devices (
    token text primary key,
    account_id uuid not null references accounts (id),
    platform text not null check (platform in ('android', 'ios')),
    registered_at timestamptz not null
);
create index push_devices_account_idx on push_devices (account_id);

create table push_settings (
    account_id uuid primary key references accounts (id),
    gameplay boolean not null,
    promotional boolean not null,
    updated_at timestamptz not null
);

-- +goose Down
drop table push_settings;
drop table push_devices;

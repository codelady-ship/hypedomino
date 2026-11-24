-- +goose Up
create table medal_ownership (
    account_id uuid not null references accounts (id),
    medal_id text not null,
    source_type text not null,
    source_ref text not null,
    granted_at timestamptz not null,
    revoked_at timestamptz,
    primary key (account_id, medal_id)
);

create table medal_showcases (
    account_id uuid primary key references accounts (id),
    medals text[] not null,
    updated_at timestamptz not null
);

-- +goose Down
drop table medal_showcases;
drop table medal_ownership;

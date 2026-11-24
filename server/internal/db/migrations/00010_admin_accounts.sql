-- +goose Up
create table admin_accounts (
    id uuid primary key default gen_random_uuid(),
    email text not null check (email = lower(btrim(email)) and email like '%_@_%'),
    display_name text not null check (btrim(display_name) <> ''),
    roles text[] not null check (cardinality(roles) > 0),
    state text not null default 'active' check (state in ('active', 'disabled')),
    created_at timestamptz not null,
    last_login_at timestamptz,
    disabled_at timestamptz
);
create unique index admin_accounts_email_idx on admin_accounts (email);

-- +goose Down
drop table admin_accounts;

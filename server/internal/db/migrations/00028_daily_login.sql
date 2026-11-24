-- +goose Up
create table daily_logins (
    account_id uuid primary key references accounts (id),
    claims bigint not null,
    last_day date not null,
    updated_at timestamptz not null
);

-- +goose Down
drop table daily_logins;

-- +goose Up
create table friend_codes (
    account_id uuid primary key references accounts (id),
    code text not null unique check (code ~ '^[0-9A-HJKMNP-TV-Z]{8}$')
);

-- +goose Down
drop table friend_codes;

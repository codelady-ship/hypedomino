-- +goose Up
create table app_attest_keys (
    key_id bytea primary key,
    account_id uuid not null references accounts (id),
    public_key bytea not null,
    counter bigint not null,
    receipt bytea not null,
    development boolean not null,
    attested_at timestamptz not null
);

create index app_attest_keys_account_idx on app_attest_keys (account_id);

-- +goose Down
drop table app_attest_keys;

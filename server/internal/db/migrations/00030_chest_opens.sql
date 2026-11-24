-- +goose Up
create table chest_opens (
    id bigserial primary key,
    account_id uuid not null references accounts (id),
    offer_id text not null,
    idempotency_key text not null,
    roll integer not null,
    pity_before integer not null,
    pity_after integer not null,
    rarity text not null,
    item_id text not null,
    config_version bigint not null,
    created_at timestamptz not null,
    unique (account_id, idempotency_key)
);

create index chest_opens_account_idx on chest_opens (account_id, id desc);

-- +goose Down
drop table chest_opens;

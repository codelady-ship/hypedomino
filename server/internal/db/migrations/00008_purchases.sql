-- +goose Up
create table purchases (
    id bigint generated always as identity primary key,
    platform text not null check (platform in ('google', 'apple')),
    transaction_id text not null,
    order_id text not null default '',
    account_id uuid not null references accounts (id),
    offer_id text not null,
    product_id text not null,
    environment text not null,
    state text not null check (state in ('pending', 'granted', 'refunded', 'cancelled')),
    grant_key text not null default '',
    bonus_key text not null default '',
    purchased_at timestamptz not null,
    created_at timestamptz not null default now(),
    granted_at timestamptz,
    refunded_at timestamptz,
    unique (platform, transaction_id)
);
create index purchases_account_idx on purchases (account_id, created_at desc);

create table purchase_notices (
    platform text not null,
    notice_id text not null,
    kind text not null,
    received_at timestamptz not null default now(),
    primary key (platform, notice_id)
);

alter table wallets drop constraint wallets_balance_check;
alter table ledger_entries drop constraint ledger_entries_balance_after_check;

-- +goose Down
alter table ledger_entries add constraint ledger_entries_balance_after_check check (balance_after >= 0);
alter table wallets add constraint wallets_balance_check check (balance >= 0);
drop table purchase_notices;
drop table purchases;

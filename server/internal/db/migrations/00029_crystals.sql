-- +goose Up
alter table ledger_entries add column currency text not null default 'coin' check (currency in ('coin', 'crystal'));
alter table ledger_entries alter column currency drop default;
alter table ledger_entries drop constraint ledger_entries_account_id_idempotency_key_key;
alter table ledger_entries add constraint ledger_entries_account_currency_key unique (account_id, currency, idempotency_key);

alter table wallets add column currency text not null default 'coin' check (currency in ('coin', 'crystal'));
alter table wallets alter column currency drop default;
alter table wallets drop constraint wallets_pkey;
alter table wallets add primary key (account_id, currency);

-- +goose Down
delete from wallets where currency = 'crystal';
alter table wallets drop constraint wallets_pkey;
alter table wallets add primary key (account_id);
alter table wallets drop column currency;

delete from ledger_entries where currency = 'crystal';
alter table ledger_entries drop constraint ledger_entries_account_currency_key;
alter table ledger_entries add constraint ledger_entries_account_id_idempotency_key_key unique (account_id, idempotency_key);
alter table ledger_entries drop column currency;

-- +goose Up
create table economy_monitor_runs (
    day date primary key,
    ran_at timestamptz not null
);

create index ledger_entries_currency_created_idx on ledger_entries (currency, created_at);

-- +goose Down
drop index ledger_entries_currency_created_idx;
drop table economy_monitor_runs;

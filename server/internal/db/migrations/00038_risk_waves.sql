-- +goose Up
alter table accounts add column shadowbanned_at timestamptz;

create table risk_reviews (
    id bigserial primary key,
    account_id uuid not null references accounts (id),
    cause text not null check (cause in ('score', 'hard_signal')),
    opened_at timestamptz not null,
    closed_at timestamptz
);

create unique index risk_reviews_open_idx on risk_reviews (account_id) where closed_at is null;

create table risk_actions (
    id bigserial primary key,
    account_id uuid not null references accounts (id),
    action text not null check (action in ('shadowban')),
    score integer not null,
    wave timestamptz not null,
    created_at timestamptz not null
);

create index risk_actions_account_idx on risk_actions (account_id, created_at desc);

create table risk_waves (
    wave timestamptz primary key,
    started_at timestamptz not null
);

-- +goose Down
drop table risk_waves;
drop table risk_actions;
drop table risk_reviews;
alter table accounts drop column shadowbanned_at;

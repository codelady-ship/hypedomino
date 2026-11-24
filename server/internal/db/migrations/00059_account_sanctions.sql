-- +goose Up
create table account_sanctions (
    id bigint generated always as identity primary key,
    account_id uuid not null references accounts (id),
    kind text not null check (kind in ('temporary_ban', 'permanent_ban', 'ranked_ban')),
    category text not null default 'other'
        check (category in ('cheating', 'collusion', 'afk_leaving', 'offensive_name', 'payment_fraud', 'other')),
    starts_at timestamptz not null,
    ends_at timestamptz,
    lifted_at timestamptz,
    check (ends_at is null or ends_at > starts_at),
    check ((kind = 'permanent_ban') = (ends_at is null))
);
create index account_sanctions_account on account_sanctions (account_id, starts_at desc, id desc);

insert into account_sanctions (account_id, kind, starts_at, ends_at)
select id, 'temporary_ban', now(), banned_until from accounts where banned_until > now();
insert into account_sanctions (account_id, kind, starts_at)
select id, 'permanent_ban', now() from accounts where state = 'banned';

-- +goose Down
drop table account_sanctions;

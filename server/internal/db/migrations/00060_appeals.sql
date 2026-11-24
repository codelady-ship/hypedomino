-- +goose Up
alter table account_sanctions drop constraint account_sanctions_kind_check;
alter table account_sanctions drop constraint account_sanctions_check1;
alter table account_sanctions add constraint account_sanctions_kind_check
    check (kind in ('temporary_ban', 'permanent_ban', 'ranked_ban', 'penalty_ban', 'forced_rename'));
alter table account_sanctions add constraint account_sanctions_ends_check
    check (case kind
        when 'permanent_ban' then ends_at is null
        when 'penalty_ban' then true
        when 'forced_rename' then true
        else ends_at is not null end);

insert into account_sanctions (account_id, kind, category, starts_at)
select account_id, 'forced_rename', 'offensive_name', now() from profiles where rename_required;
insert into account_sanctions (account_id, kind, category, starts_at, ends_at)
select account_id, 'penalty_ban', 'afk_leaving', now(),
    case when (record -> 'Ranked' ->> 'UnderReview')::boolean then null else (record -> 'Ranked' ->> 'BanUntil')::timestamptz end
from penalty_records
where coalesce((record -> 'Ranked' ->> 'UnderReview')::boolean, false) or (record -> 'Ranked' ->> 'BanUntil')::timestamptz > now();

create table appeals (
    id bigint generated always as identity primary key,
    account_id uuid not null references accounts (id),
    sanction_id bigint not null unique references account_sanctions (id),
    text text not null check (btrim(text) <> ''),
    state text not null default 'open' check (state in ('open', 'upheld', 'lifted')),
    created_at timestamptz not null,
    decided_at timestamptz,
    decided_by text,
    check ((state = 'open') = (decided_at is null))
);
create index appeals_open on appeals (id) where state = 'open';
create index appeals_account on appeals (account_id, created_at desc);

-- +goose Down
drop table appeals;
delete from account_sanctions where kind in ('penalty_ban', 'forced_rename');
alter table account_sanctions drop constraint account_sanctions_ends_check;
alter table account_sanctions drop constraint account_sanctions_kind_check;
alter table account_sanctions add constraint account_sanctions_kind_check check (kind in ('temporary_ban', 'permanent_ban', 'ranked_ban'));
alter table account_sanctions add constraint account_sanctions_check1 check ((kind = 'permanent_ban') = (ends_at is null));

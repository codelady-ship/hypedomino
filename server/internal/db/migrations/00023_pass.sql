-- +goose Up
create table pass_progress (
    account_id uuid not null references accounts (id),
    season_id text not null,
    xp_total bigint not null default 0 check (xp_total >= 0),
    premium_ref text,
    premium_at timestamptz,
    entered_at timestamptz not null,
    primary key (account_id, season_id),
    check ((premium_ref is null) = (premium_at is null))
);

create index pass_progress_season on pass_progress (season_id, account_id);

create table pass_season_ends (
    season_id text primary key,
    config_version bigint not null,
    accounts integer not null check (accounts >= 0),
    closed_at timestamptz not null
);

-- +goose Down
drop table pass_season_ends;
drop table pass_progress;

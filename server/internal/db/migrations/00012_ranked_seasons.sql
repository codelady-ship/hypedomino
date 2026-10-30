-- +goose Up
create table ranked_seasons (
    id text primary key,
    name_key text not null default '',
    starts_at timestamptz not null,
    ends_at timestamptz not null check (ends_at > starts_at),
    closed_1v1_at timestamptz,
    closed_2v2_at timestamptz,
    created_at timestamptz not null,
    exclude using gist (tstzrange(starts_at, ends_at) with &&)
);

alter table ratings
    add column season_id text references ranked_seasons (id),
    add column reset_index smallint check (reset_index between 0 and 27);
alter table rating_results add column season_id text references ranked_seasons (id);

create table season_results (
    season_id text not null references ranked_seasons (id),
    mode text not null check (mode in ('1v1', '2v2')),
    account_id uuid not null references accounts (id),
    position integer not null check (position >= 1),
    rank_index smallint not null,
    apex_tier text,
    lp integer not null,
    wins integer not null,
    mmr integer not null,
    reset_index smallint not null,
    primary key (season_id, mode, account_id),
    unique (season_id, mode, position)
);

-- +goose Down
drop table season_results;
alter table rating_results drop column season_id;
alter table ratings drop column season_id, drop column reset_index;
drop table ranked_seasons;

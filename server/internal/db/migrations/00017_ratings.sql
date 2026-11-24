-- +goose Up
create table ratings (
    account_id uuid not null references accounts (id),
    mode text not null check (mode in ('1v1', '2v2')),
    mmr integer not null,
    matches integer not null default 0,
    placement_played integer not null default 0,
    placement_penalty integer not null default 0,
    placed boolean not null default false,
    rank_index smallint not null default 0 check (rank_index between 0 and 28),
    lp integer not null default 0 check (lp >= 0),
    updated_at timestamptz not null,
    primary key (account_id, mode)
);
create index ratings_mode_rank_idx on ratings (mode, rank_index desc, lp desc) where placed;

create table rating_results (
    match_id uuid primary key,
    mode text not null check (mode in ('1v1', '2v2')),
    config_version bigint not null,
    applied_at timestamptz not null
);

create table rating_changes (
    match_id uuid not null references rating_results (match_id),
    account_id uuid not null references accounts (id),
    won boolean not null,
    offender boolean not null,
    placement boolean not null,
    k smallint not null,
    expected real not null,
    mmr_before integer not null,
    mmr_after integer not null,
    lp_delta integer not null,
    rank_before smallint not null,
    lp_before integer not null,
    rank_after smallint not null,
    lp_after integer not null,
    primary key (match_id, account_id)
);
create index rating_changes_account_idx on rating_changes (account_id, match_id);

-- +goose Down
drop table rating_changes;
drop table rating_results;
drop table ratings;

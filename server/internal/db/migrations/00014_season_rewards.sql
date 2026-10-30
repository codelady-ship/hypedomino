-- +goose Up
alter table ranked_seasons
    add column rewards boolean not null default true,
    add column rewarded_1v1_at timestamptz,
    add column rewarded_2v2_at timestamptz;

create table season_rewards (
    season_id text not null references ranked_seasons (id),
    mode text not null check (mode in ('1v1', '2v2')),
    account_id uuid not null references accounts (id),
    tier text not null,
    rank_index smallint not null,
    coins bigint not null,
    bundle_id text,
    grant_id bigint references reward_grants (id),
    emblems text[] not null default '{}',
    created_at timestamptz not null,
    seen_at timestamptz,
    primary key (season_id, mode, account_id)
);
create index season_rewards_unseen_idx on season_rewards (account_id) where seen_at is null;

create table leaderboard_emblems (
    account_id uuid not null references accounts (id),
    mode text not null check (mode in ('1v1', '2v2')),
    season_id text not null references ranked_seasons (id),
    board text not null,
    bracket integer not null check (bracket >= 1),
    position integer not null check (position >= 1),
    awarded_at timestamptz not null,
    revoked_at timestamptz,
    revoke_reason text,
    primary key (account_id, mode, season_id, board)
);

-- +goose Down
drop table leaderboard_emblems;
drop table season_rewards;
alter table ranked_seasons drop column rewarded_2v2_at, drop column rewarded_1v1_at, drop column rewards;

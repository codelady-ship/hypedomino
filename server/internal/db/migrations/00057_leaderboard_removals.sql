-- +goose Up
create table leaderboard_removals (
    season_id text not null references ranked_seasons (id),
    mode text not null check (mode in ('1v1', '2v2')),
    account_id uuid not null references accounts (id),
    removed_at timestamptz not null,
    primary key (season_id, mode, account_id)
);

-- +goose Down
drop table leaderboard_removals;

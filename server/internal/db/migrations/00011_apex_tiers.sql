-- +goose Up
alter table ratings
    add column apex_tier text check (apex_tier in ('master', 'grandmaster', 'challenger')),
    add column lp_at timestamptz,
    add column wins integer not null default 0;
create index ratings_apex_idx on ratings (mode, lp desc, lp_at, wins desc, account_id) where rank_index = 28;

-- +goose Down
drop index ratings_apex_idx;
alter table ratings drop column apex_tier, drop column lp_at, drop column wins;

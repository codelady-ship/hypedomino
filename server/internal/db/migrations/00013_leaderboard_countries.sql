-- +goose Up
alter table accounts add column country_changed_at timestamptz;
alter table ratings add column board_country char(2);
alter table season_results add column country char(2);
create index ratings_country_idx on ratings (mode, season_id, board_country) where placed;

-- +goose Down
drop index ratings_country_idx;
alter table season_results drop column country;
alter table ratings drop column board_country;
alter table accounts drop column country_changed_at;

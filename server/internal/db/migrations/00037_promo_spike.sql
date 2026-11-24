-- +goose Up
alter table promo_codes add column spike_at timestamptz;

-- +goose Down
alter table promo_codes drop column spike_at;

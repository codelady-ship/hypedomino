-- +goose Up
alter table promo_codes add column pass_premium boolean not null default false;

-- +goose Down
alter table promo_codes drop column pass_premium;

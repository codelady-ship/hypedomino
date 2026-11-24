-- +goose Up
alter table promo_codes add column batch_id text not null default '';
create index promo_codes_batch_idx on promo_codes (batch_id) where batch_id <> '';
create index promo_codes_campaign_idx on promo_codes (campaign_id, created_at desc);

-- +goose Down
drop index promo_codes_campaign_idx;
drop index promo_codes_batch_idx;
alter table promo_codes drop column batch_id;

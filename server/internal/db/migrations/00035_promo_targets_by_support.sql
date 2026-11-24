-- +goose Up
alter table promo_codes drop constraint promo_codes_cap_check;
alter table promo_codes add constraint promo_codes_cap_check check (cap >= 1 or (kind = 'targeted' and cap >= 0));

-- +goose Down
delete from promo_redemptions where promo_id in (select promo_id from promo_codes where cap = 0);
delete from promo_targets where promo_id in (select promo_id from promo_codes where cap = 0);
delete from promo_codes where cap = 0;
alter table promo_codes drop constraint promo_codes_cap_check;
alter table promo_codes add constraint promo_codes_cap_check check (cap >= 1);

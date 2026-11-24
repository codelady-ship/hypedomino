-- +goose Up
alter table admin_accounts add column locale text check (locale in ('az', 'en'));

-- +goose Down
alter table admin_accounts drop column locale;

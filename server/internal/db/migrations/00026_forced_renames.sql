-- +goose Up
alter table profiles add column rename_required boolean not null default false;

-- +goose Down
alter table profiles drop column rename_required;

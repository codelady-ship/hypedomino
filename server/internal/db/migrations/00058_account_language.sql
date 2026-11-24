-- +goose Up
alter table accounts add column language text;

-- +goose Down
alter table accounts drop column language;

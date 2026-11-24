-- +goose Up
alter table accounts add column muted_until timestamptz;

-- +goose Down
alter table accounts drop column muted_until;

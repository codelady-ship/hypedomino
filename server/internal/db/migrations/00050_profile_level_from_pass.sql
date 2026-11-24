-- +goose Up
alter table profiles drop column level, drop column xp;

-- +goose Down
alter table profiles
    add column level integer not null default 1 check (level >= 1),
    add column xp bigint not null default 0 check (xp >= 0);

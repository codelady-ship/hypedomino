-- +goose Up
create table config_snapshots (
    version bigint generated always as identity primary key,
    domain text not null,
    content jsonb not null,
    published_at timestamptz not null default now()
);
create index config_snapshots_domain_idx on config_snapshots (domain, version desc);

-- +goose Down
drop table config_snapshots;

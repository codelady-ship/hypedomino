-- +goose Up
create table analytics_ids (
    analytics_id uuid primary key,
    account_id uuid not null references accounts (id),
    first_seen_at timestamptz not null
);
create index analytics_ids_account_idx on analytics_ids (account_id);

create table analytics_events (
    id bigint generated always as identity primary key,
    analytics_id uuid not null references analytics_ids (analytics_id) on delete cascade,
    name text not null,
    label text not null,
    value bigint not null,
    client_time_ms bigint not null,
    received_at timestamptz not null
);
create index analytics_events_analytics_idx on analytics_events (analytics_id);
create index analytics_events_received_idx on analytics_events (received_at);

-- +goose Down
drop table analytics_events;
drop table analytics_ids;

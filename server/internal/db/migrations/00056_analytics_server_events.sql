-- +goose Up
alter table analytics_events alter column analytics_id drop not null, alter column client_time_ms drop not null;

create table analytics_sources (
    topic text not null,
    key text not null,
    recorded_at timestamptz not null,
    primary key (topic, key)
);
create index analytics_sources_recorded_idx on analytics_sources (recorded_at);

-- +goose Down
drop table analytics_sources;
delete from analytics_events where analytics_id is null or client_time_ms is null;
alter table analytics_events alter column analytics_id set not null, alter column client_time_ms set not null;

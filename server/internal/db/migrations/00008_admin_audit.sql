-- +goose Up
create table admin_audit_entries (
    id bigint generated always as identity primary key,
    created_at timestamptz not null,
    actor_id text not null,
    actor_roles text[] not null,
    environment text not null check (environment in ('staging', 'production')),
    action text not null,
    object_type text not null,
    object_id text not null,
    before jsonb,
    after jsonb,
    reason text not null default '',
    approval_id bigint references admin_audit_entries (id),
    ref text not null default '',
    change_set text not null default '',
    config_version bigint,
    session_id text not null,
    source_ip inet,
    user_agent text not null default ''
);
create index admin_audit_entries_time_idx on admin_audit_entries (created_at desc, id desc);
create index admin_audit_entries_object_idx on admin_audit_entries (object_type, object_id, id desc);
create index admin_audit_entries_actor_idx on admin_audit_entries (actor_id, id desc);

-- +goose StatementBegin
create function admin_audit_entries_append_only() returns trigger language plpgsql as $$
begin
    raise exception 'admin_audit_entries is append-only';
end;
$$;
-- +goose StatementEnd

create trigger admin_audit_entries_append_only
    before update or delete or truncate on admin_audit_entries
    for each statement execute function admin_audit_entries_append_only();

-- +goose Down
drop table admin_audit_entries;
drop function admin_audit_entries_append_only();

-- +goose Up
create table game_strings (
    key text primary key,
    note text not null,
    max_length integer not null check (max_length > 0),
    placeholders jsonb not null default '[]',
    deprecated boolean not null default false,
    updated_by text not null,
    updated_at timestamptz not null
);

create table game_string_values (
    key text not null references game_strings (key),
    language text not null check (language in ('az', 'en', 'tr', 'ru')),
    text text not null,
    status text not null check (status in ('draft', 'approved')),
    updated_by text not null,
    updated_at timestamptz not null,
    primary key (key, language)
);

create table game_string_bundles (
    language text not null check (language in ('az', 'en', 'tr', 'ru')),
    version integer not null check (version > 0),
    hash text not null,
    content bytea not null,
    published_by text not null,
    published_at timestamptz not null,
    primary key (language, version),
    unique (language, hash)
);

-- +goose StatementBegin
create function game_string_bundles_append_only() returns trigger language plpgsql as $$
begin
    raise exception 'game_string_bundles is append-only';
end;
$$;
-- +goose StatementEnd

create trigger game_string_bundles_append_only
    before update or delete or truncate on game_string_bundles
    for each statement execute function game_string_bundles_append_only();

-- +goose Down
drop table game_string_bundles;
drop function game_string_bundles_append_only();
drop table game_string_values;
drop table game_strings;

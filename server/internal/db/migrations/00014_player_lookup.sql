-- +goose Up
create index auth_identities_email_idx on auth_identities (lower(email));
create index profiles_nickname_prefix_idx on profiles (nickname_key text_pattern_ops);

-- +goose Down
drop index profiles_nickname_prefix_idx;
drop index auth_identities_email_idx;

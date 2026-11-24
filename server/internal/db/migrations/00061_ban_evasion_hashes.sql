-- +goose Up
create table ban_evasion_hashes (
    identity_hash bytea primary key check (length(identity_hash) = 32),
    category text not null
        check (category in ('cheating', 'collusion', 'afk_leaving', 'offensive_name', 'payment_fraud', 'other')),
    ban_ends_at timestamptz,
    expires_at timestamptz not null,
    check (ban_ends_at is null or ban_ends_at = expires_at)
);
create index ban_evasion_hashes_expiry on ban_evasion_hashes (expires_at);

-- +goose Down
drop table ban_evasion_hashes;

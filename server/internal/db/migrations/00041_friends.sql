-- +goose Up
create table friendships (
    account_id uuid not null references accounts (id),
    friend_id uuid not null references accounts (id),
    created_at timestamptz not null,
    primary key (account_id, friend_id),
    check (account_id <> friend_id)
);

create table friend_requests (
    sender_id uuid not null references accounts (id),
    recipient_id uuid not null references accounts (id),
    created_at timestamptz not null,
    primary key (sender_id, recipient_id),
    check (sender_id <> recipient_id)
);
create index friend_requests_recipient_idx on friend_requests (recipient_id);

create table blocks (
    account_id uuid not null references accounts (id),
    blocked_id uuid not null references accounts (id),
    created_at timestamptz not null,
    primary key (account_id, blocked_id),
    check (account_id <> blocked_id)
);
create index blocks_blocked_idx on blocks (blocked_id);

-- +goose Down
drop table blocks;
drop table friend_requests;
drop table friendships;

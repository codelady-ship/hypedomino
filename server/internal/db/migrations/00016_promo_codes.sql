-- +goose Up
create table promo_codes (
    id bigint generated always as identity primary key,
    promo_id text not null unique,
    campaign_id text not null,
    kind text not null check (kind in ('single_use', 'multi_use', 'targeted')),
    bundle jsonb not null,
    code_hash bytea not null unique,
    starts_at timestamptz not null,
    ends_at timestamptz not null,
    cap integer not null check (cap >= 1),
    state text not null check (state in ('pending_approval', 'active', 'disabled')),
    created_by text not null,
    approved_by text not null default '',
    created_at timestamptz not null default now(),
    disabled_at timestamptz
);

create table promo_targets (
    promo_id text not null references promo_codes (promo_id),
    account_id uuid not null references accounts (id),
    primary key (promo_id, account_id)
);

create table promo_redemptions (
    promo_id text not null references promo_codes (promo_id),
    account_id uuid not null references accounts (id),
    grant_key text not null,
    redeemed_at timestamptz not null,
    primary key (promo_id, account_id)
);

create table promo_attempts (
    id bigint generated always as identity primary key,
    account_id uuid not null references accounts (id),
    ok boolean not null,
    attempted_at timestamptz not null
);
create index promo_attempts_account_idx on promo_attempts (account_id, attempted_at desc);

-- +goose Down
drop table promo_attempts;
drop table promo_redemptions;
drop table promo_targets;
drop table promo_codes;

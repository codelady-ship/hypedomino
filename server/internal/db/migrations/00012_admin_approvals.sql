-- +goose Up
create table admin_approvals (
    id bigint generated always as identity primary key,
    kind text not null,
    object_type text not null,
    object_id text not null,
    change jsonb not null,
    reason text not null check (btrim(reason) <> ''),
    proposer_id text not null,
    proposer_roles text[] not null check (cardinality(proposer_roles) > 0),
    proposed_at timestamptz not null,
    expires_at timestamptz not null check (expires_at > proposed_at),
    state text not null default 'pending' check (state in ('pending', 'approved', 'rejected')),
    decider_id text,
    decider_roles text[],
    decided_at timestamptz,
    check ((state = 'pending') = (decider_id is null)),
    check (decider_id <> proposer_id)
);
create index admin_approvals_pending_idx on admin_approvals (expires_at) where state = 'pending';

-- +goose Down
drop table admin_approvals;

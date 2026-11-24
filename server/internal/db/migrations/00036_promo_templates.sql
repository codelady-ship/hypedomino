-- +goose Up
create table promo_templates (
    promo_id text primary key,
    campaign_id text not null,
    kind text not null check (kind in ('single_use', 'multi_use', 'targeted')),
    bundle jsonb,
    created_by text not null,
    created_at timestamptz not null,
    updated_at timestamptz not null
);

-- +goose Down
drop table promo_templates;

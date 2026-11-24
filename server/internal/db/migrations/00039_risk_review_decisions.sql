-- +goose Up
alter table accounts add column risk_reviewed_at timestamptz;

alter table risk_reviews
    add column decision text check (decision in ('clear', 'confirm')),
    add column decided_by text;

-- +goose Down
alter table risk_reviews drop column decided_by, drop column decision;
alter table accounts drop column risk_reviewed_at;

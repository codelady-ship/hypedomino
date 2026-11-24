-- +goose Up
alter table match_participants add column partner_decisions integer check (partner_decisions >= 0);
alter table match_participants add column partner_fits integer check (partner_fits >= 0 and partner_fits <= partner_decisions);

-- +goose Down
alter table match_participants drop column partner_fits;
alter table match_participants drop column partner_decisions;

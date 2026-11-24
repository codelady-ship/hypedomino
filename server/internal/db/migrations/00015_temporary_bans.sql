-- +goose Up
alter table accounts add column banned_until timestamptz;
alter table auth_sessions drop constraint auth_sessions_revoke_reason_check;
alter table auth_sessions add constraint auth_sessions_revoke_reason_check
    check (revoke_reason in ('logout', 'logout_all', 'reuse', 'device', 'expired', 'banned'));

-- +goose Down
update auth_sessions set revoke_reason = 'logout_all' where revoke_reason = 'banned';
alter table auth_sessions drop constraint auth_sessions_revoke_reason_check;
alter table auth_sessions add constraint auth_sessions_revoke_reason_check
    check (revoke_reason in ('logout', 'logout_all', 'reuse', 'device', 'expired'));
alter table accounts drop column banned_until;

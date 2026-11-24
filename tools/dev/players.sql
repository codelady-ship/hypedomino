insert into accounts (id, country)
select ('00000000-0000-4000-8000-' || lpad(n::text, 12, '0'))::uuid, 'AZ'
from generate_series(1, 8) n
on conflict (id) do nothing;

insert into auth_identities (provider, subject, account_id, email)
select 'google', 'dev-player-' || n, ('00000000-0000-4000-8000-' || lpad(n::text, 12, '0'))::uuid, 'dev-player-' || n || '@example.com'
from generate_series(1, 8) n
on conflict (provider, subject) do nothing;

insert into profiles (account_id, nickname, nickname_key)
select ('00000000-0000-4000-8000-' || lpad(n::text, 12, '0'))::uuid, 'DevPlayer' || n, 'devplayer' || n
from generate_series(1, 8) n
on conflict (account_id) do nothing;

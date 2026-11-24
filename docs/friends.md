# Friends and blocking

Scope: friend requests, the friends list, removing a friend and blocking (#153). In-game lobby invites follow in #278 (`docs/friend-lobby.md` FL-1). Owner decisions of 2026-11-08.

## 1. Requests

- **FR-1** A friendship is mutual. One player sends a request; the other accepts or declines it. Until then nothing is shared.
- **FR-2** A request names the other player either by nickname, matched by the same normalised key that keeps nicknames unique (`server/internal/profile`), or by friend code, never both.
- **FR-2a** Every account has one friend code, created on first request and never changed: 8 characters of Crockford base32 (digits and capitals without I, L, O, U), shown as `XXXX-XXXX`. Input ignores case, spaces and the dash, and reads O as 0 and I or L as 1.
- **FR-3** A request to oneself is refused. A request to a player who is already a friend is refused.
- **FR-4** A request to a player who already sent one to the sender makes both friends at once.
- **FR-5** Repeating an open request changes nothing.
- **FR-6** A player has at most `FriendRequestsMax` open outgoing requests and receives at most the same number of open incoming requests. A request over either limit is refused with `friend.requests_full`.
- **FR-7** The recipient accepts or declines; the sender may cancel. A declined or cancelled request leaves no trace, and the sender is not told it was declined.

## 2. Friends list

- **FR-8** A player has at most `FriendsMax` friends. Accepting a request when either side is at the limit is refused with `friend.friends_full`.
- **FR-9** Either friend can remove the friendship at any time. The other side is not notified.
- **FR-10** The list shows friends, incoming and outgoing requests and blocked players with account id and nickname. Accounts that are not active (pending deletion, deleted, permanently banned) do not appear in friends or requests (`docs/privacy.md` 8.2).

- **FR-17** The friends leaderboard is the global ranked order of a mode and season filtered to the player and their active friends (`docs/ranking.md` 8.2): `GET /leaderboard/{mode}?board=friends`. Friends who are not placed in that season and mode are not on it. It awards no emblems.

## 2a. Presence

- **FR-15** Only friends see a player's presence (`docs/privacy.md` 5): `online` while the app holds a game connection, `in_match` while that connection has a seat in a match, `offline` otherwise. The mode, room and opponents are not shown.
- **FR-16** Each game node writes the presence of its connected accounts to Redis every `HD_PING_INTERVAL` with a lifetime of three intervals, so a closed app reads `offline` within three intervals. Nothing about presence is stored in PostgreSQL. Without Redis every friend reads `offline`.

## 3. Blocking

- **FR-11** Blocking removes any friendship and every open request between the two players in both directions.
- **FR-12** While either player blocks the other, neither can send the other a friend request or a lobby invite (`docs/friend-lobby.md` FL-1). A request to a player who blocked the sender is refused as if the nickname were unknown, so the blocked player does not learn about the block.
- **FR-13** Blocking has no effect on matchmaking.
- **FR-14** Unblocking restores nothing; the players start again from a request.

## 4. API

Player HTTP API with the access token (`server/internal/friends`):

| Request | Effect |
|---|---|
| `GET /friends/code` | The player's own friend code `{code: "XXXX-XXXX"}` |
| `GET /friends` | `friends`, `incoming`, `outgoing`, `blocked`, each a list of `{account_id, nickname, since}`; friends also carry `presence` (FR-15) |
| `POST /friends/requests` `{nickname}` or `{code}` | Sends a request; returns `{status: "requested"}` or `{status: "friends"}` (FR-4) |
| `POST /friends/requests/{account_id}/accept` | Accepts the request of that account |
| `POST /friends/requests/{account_id}/decline` | Declines it |
| `DELETE /friends/requests/{account_id}` | Cancels the sender's own request to that account |
| `DELETE /friends/{account_id}` | Removes the friendship |
| `PUT /friends/blocks/{account_id}` | Blocks that account |
| `DELETE /friends/blocks/{account_id}` | Unblocks it |

Refusals: `friend.unknown_player` (404), `friend.self`, `friend.target_required` (400), `friend.already_friends`, `friend.friends_full`, `friend.requests_full` (409), `friend.no_request`, `friend.not_friends` (404).

## 5. Config

| Name | Meaning | Launch default |
|---|---|---|
| `FriendsMax` (`HD_FRIENDS_MAX`) | Friends per account | 100 |
| `FriendRequestsMax` (`HD_FRIEND_REQUESTS_MAX`) | Open outgoing, and open incoming, requests per account | 50 |

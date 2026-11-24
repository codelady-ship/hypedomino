# Friend lobby and custom matches

How friends play together: the friend lobby, team setup and the start of a custom match (R-55). Ranked and casual matches never use any of this: matchmaking forms the match and it opens straight into the loading screen (R-53, R-57). Owner decision 2026-10-28.

Owned elsewhere and only referenced: game rules of custom play (`docs/rules.md` R-55, R-57, R-58), friends and blocking (#153), emoji ownership and limits (#150, #151), avatars and frames (`docs/cosmetics.md`), the loading screen (#85).

## 1. Flow

| Stage | What the players see | Who moves it on |
|---|---|---|
| FL-1 Lobby | Every member's equipped avatar in its frame with nickname and rank badge, the leader marked, the invite panel and emoji | The creator, with Start |
| FL-2 Team setup | Two team columns with the seats of the chosen mode, the friends panel to invite into free seats | The creator, with Start match |
| FL-3 Loading | The loading screen of every match (R-57, #85) | The server, when everyone is ready or `MatchLoadingSeconds` have passed |
| FL-4 Match | The match scene | The match end |
| FL-5 Back to lobby | Every member who is still in the room returns to FL-1 with the result shown | Automatic |

A member can leave the room in FL-1 and FL-2. Leaving during FL-3 and FL-4 is a leave under R-52 with no penalty (R-55).

FL-5 follows the server: when the custom match ends, the server puts the room back in the lobby phase, and the client of every member still in the room returns to it from the match on that `LobbyState`. A member who left the room during the match returns to the main lobby (`client/README.md`, `AppFlow.Room`).

## 2. Lobby (FL-1)

- Members: the creator plus invited friends, at most 4, the seat count of the largest mode (R-5). The room has a code for joining without an invite (#268).
- Members panel: each member shows as their equipped avatar inside their equipped frame, animated where the effect level allows and the static fallback otherwise, with nickname and rank badge. There is no 3D scene and no character (owner decision 2026-11-12).
- Invite: from the friends panel (#153), an online friend gets an in-game invite; anyone else gets a link that opens the game into the room (deep link) or the room code. Until the link format is decided (#154), the panel offers the room code only. A friend in a match or already in the room is not invited, and the panel sends one invite per friend until the room state changes (`client/README.md`, Friend lobby, `InvitePanel`). A blocked player cannot invite or be invited (#153).
- Emoji: a member sends an owned emoji (#150); it shows over their avatar for every member. The server refuses an emoji the member does not own and applies the same spam limit as in a match (#151).
- Leadership: the creator controls the room. When the creator leaves, control passes to the member who joined earliest; the room closes when the last member leaves.

## 3. Team setup (FL-2)

- The creator picks the mode, 1v1 or 2v2, and places members into seats: N and S are one team, E and W the other (R-6); X and Y in 1v1 (R-8). Members may ask for a seat by tapping it; the creator decides (R-55).
- Free seats can still be filled by invite from the friends panel. Members who get no seat in 1v1 wait in the lobby until the match ends and see its result.
- Start match needs every seat of the mode filled (#268). Then the match opens with the loading phase (FL-3).
- The creator can go back to the lobby from team setup; seats are kept.

## 4. What the server decides

- Membership, leadership, seats, mode, the start and the return to the lobby are server state; the client sends intents only (CLAUDE.md, zero trust).
- Emoji are relayed, not trusted: ownership is checked on every intent.
- A member whose match has not ended cannot join another room or queue (R-58).
- Nothing in a custom room grants coins, pass XP, missions, medals or rating (R-55, `docs/economy.md` 2.1).
- Protocol: `LobbyIntent`, `LobbyState`, `LobbyRelay` and `LobbyInvite` (`proto/README.md`). A member's emoji and mute come from their loadout when they create or join the room. The lobby has no actions: `LOBBY_COMMAND_ACT` and the `action` fields are deprecated and unused, and an `ACT` intent is refused like any unknown command. An in-game invite goes only to a friend. A member who disconnects outside a match leaves the room; during a custom match they keep their place until the reconnect grace ends. A seat left by a disconnected or removed player is played by the bot at the casual bot level of the matchmaking config (R-52, R-55).
- Rooms live on the node that created them, and the room code is unique across nodes. A player connected to another node who joins by code is moved to the room's node: the server closes the connection, the client reconnects as after any drop and arrives in the room with its `LobbyState`. In-game invites reach the friend on whichever node they are connected to (`server/README.md`, Cluster).
- The leader adds a member who has no seat to a free seat in team setup or during the match (`ADD`, R-55); during the match the member takes the seat back from the bot. A player outside the room joins it first; a removed player can join again only in the lobby or team setup, because a running match accepts no joins.

### Refusal texts

A refused command comes back as the `LobbyRefusal` of the `LobbyState`, and the screen shows its text. The built-in bundle (`docs/localization.md` 2.4) ships these `az` and `en` texts; a published bundle replaces them from admin, and `tr` and `ru` fall back along `docs/localization.md` 1.2 until translated. A typed code the client refuses before sending (`client/README.md`, `RoomCode`) shows the `UNKNOWN_CODE` text, since no room can have it.

| Refusal | Key | `az` | `en` |
|---|---|---|---|
| `UNAVAILABLE` | `lobby.room.refused.unavailable` | Otaq hazırda əlçatan deyil. Sonra yenidən yoxla | The room is not available right now. Try again later |
| `UNKNOWN_CODE` | `lobby.room.refused.unknown_code` | Bu kodla otaq yoxdur | There is no room with this code |
| `NOT_LEADER` | `lobby.room.refused.not_leader` | Bunu yalnız otaq lideri edə bilər | Only the room leader can do this |
| `SEAT_TAKEN` | `lobby.room.refused.seat_taken` | Bu oturacaq artıq tutulub | This seat is already taken |
| `NOT_IN_LOBBY` | `lobby.room.refused.not_in_lobby` | Bu oyunçu otaqda deyil | This player is not in the room |
| `ALREADY_IN_LOBBY` | `lobby.room.refused.already_in_lobby` | Oyunçu artıq otaqdadır | The player is already in a room |
| `FULL` | `lobby.room.refused.full` | Otaq doludur | The room is full |
| `SEATS_OPEN` | `lobby.room.refused.seats_open` | Matçı başlatmaq üçün bütün oturacaqları doldur | Fill every seat to start the match |
| `STARTED` | `lobby.room.refused.started` | Matç artıq başlayıb | The match has already started |
| `INVALID_SEAT` | `lobby.room.refused.invalid_seat` | Bu oturacaq seçilmiş rejimdə yoxdur | This seat is not in the chosen mode |
| `INVALID_MODE` | `lobby.room.refused.invalid_mode` | Bu rejimi seçmək olmur | This mode cannot be chosen |
| `IN_MATCH` | `lobby.room.refused.in_match` | Oyunçunun bitməmiş matçı var | The player has a match that has not ended |
| `WRONG_PHASE` | `lobby.room.refused.wrong_phase` | Bu, hazırda mümkün deyil | This is not possible right now |
| `BLOCKED` | `lobby.room.refused.blocked` | Bu oyunçunu dəvət etmək olmur | This player cannot be invited |
| `NOT_FRIEND` | `lobby.room.refused.not_friend` | Yalnız dostları dəvət etmək olar | Only friends can be invited |
| `RANKED_BAN` | `lobby.room.refused.ranked_ban` | Reytinq oyunlarından ban olunmusan, ona görə custom otaqlar sənin üçün bağlıdır. Casual matçlar açıqdır | You are banned from ranked play, so custom rooms are closed for you. Casual matches stay open |
| Any other refusal, including the deprecated `UNKNOWN_ACTION` | `notice.generic` | Nəsə alınmadı. Yenidən cəhd et | Something went wrong. Try again |

## 5. Decisions and open items

- Decided 2026-11-12, replacing the 2026-10-28 decision on lobby actions: the lobby has no 3D characters and no actions, and the `action` cosmetic type is removed (`docs/cosmetics.md`). Members show as avatars with frames.
- Decided 2026-10-28: no bot seats in custom matches. A 2v2 needs four players; with fewer, the room plays 1v1.

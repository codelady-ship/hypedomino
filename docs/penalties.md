# AFK and leaver penalties

Penalties for players who stop playing a match (R-51) or leave it (R-52). The server alone detects offenses, applies penalties and keeps the record; the client only shows them. Every count, weight, percentage and duration below is admin config (section 8). This document fixes the structure and the order of escalation, not the values.

## 1. Scope

- Ranked 1v1 and 2v2 matches (R-53): offenses of section 2, the offense level of section 3, the ladder of section 4 and the match result of section 5.
- Casual 1v1 and 2v2 matches (R-56): offenses of section 2 on the separate, lighter casual ladder of section 6.
- Does not apply to custom rooms (R-55): nothing is recorded and nothing is penalised there, including a player removed by the room creator.
- A match that is cancelled by the server (server fault, region outage) records no offense for anyone.

## 2. Offenses

| Offense | Detected when | Weight (config) |
|---|---|---|
| AFK, returned | The player is marked AFK (R-51), takes back control by reconnecting and is in control when the match ends | `AfkReturnedWeight` |
| AFK, not returned | The player is marked AFK (R-51) and the bot is still playing for them when the match ends | `AfkWeight` |
| Leave, returned | The player leaves the match (R-52), quitting from the match screen or disconnecting, stays away longer than `ReconnectGraceSeconds`, comes back and is in control when the match ends | `LeaveReturnedWeight` |
| Leave, not returned | The player leaves the match (R-52), stays away longer than `ReconnectGraceSeconds` and the bot is still playing for them when the match ends | `LeaveWeight` |

- One match records at most one offense per player, the heaviest one. The room reports per seat at match end (`rules.PresenceReport`): total timeouts, whether the player was marked AFK, whether they left, and whether the bot was in control at the end. #52 maps the report to the offense of this table with the heaviest configured weight.
- Time away shorter than `ReconnectGraceSeconds`, after which the player plays on, is not an offense, whether they quit from the match screen or lost the connection. Turns missed while away still count as timeouts toward R-51.
- A player who is not ready when the loading phase ends (R-57) is treated as disconnected from that moment: the same grace and offenses apply as for a drop during play.
- The server cannot tell a dropped network from a deliberate quit, so it does not try: what counts is whether the player came back in time. Repeated drops are handled by the ladder like any other offense.

## 3. Offense level

- Every offense stays active for `OffenseWindowDays` from the match end and then expires.
- `CleanMatchesToForgive` completed matches without an offense in a row remove the oldest active offense early.
- A player's offense level is the sum of the weights of their active offenses. The level picks the step of the ladder.

## 4. Penalty ladder

Each step applies from the moment the offense is recorded. A higher step includes everything in the steps below it.

| Step | From level (config) | Penalty |
|---|---|---|
| 1. Warning | `WarningLevel` | A notice after the match: what was detected, the current level and the next step |
| 2. Queue delay | `QueueDelayLevel` | Every matchmaking entry waits `QueueDelaySeconds` before the search starts, for the next `QueueDelayMatches` completed matches |
| 3. Matchmaking ban | `BanLevel` | No matchmaking for `BanHours`. The duration is multiplied by `BanEscalationFactor` for each further ban inside the window. Custom rooms stay open |
| 4. Review | `ReviewLevel` | The account is flagged for manual review in the admin panel; the ban from step 3 stays until an admin decides |

## 5. Match result for the offender and the others

| Who | Rating | Rewards |
|---|---|---|
| Offender | The match counts as a loss whatever the result, plus an extra rating loss of `OffenseRatingPenalty` | No coins, pass XP, mission or medal progress from that match. AFK, returned and Leave, returned: rewards are kept, the rating penalty still applies |
| Partner of the offender (2v2), if their team loses | The rating loss is reduced by `PartnerLossReductionPercent` | Normal |
| Opponents | Normal result | Normal |

- When both partners of a team commit an offense, neither gets the partner reduction.
- When the offender's team wins (the bot and the partner finish the match), the win still counts as a loss for the offender only.

## 6. Casual matches

Casual offenses are kept apart from ranked ones. They never add to the offense level of section 3 and never lead to a matchmaking ban or a review.

- Offenses and their weights are those of section 2.
- A casual offense stays active for `CasualOffenseWindowDays` from the match end. `CasualCleanMatchesToForgive` completed casual matches without an offense in a row remove the oldest active casual offense early.
- The casual offense level is the sum of the weights of the active casual offenses. It picks the step of the casual ladder.
- Each step applies from the moment the offense is recorded. A queue delay step includes the warning.

| Step | From casual level (config) | Penalty |
|---|---|---|
| 1. Warning | `CasualWarningLevel` | A notice after the match: what was detected, the current casual level and the next step |
| 2. Casual queue delay `n` | Entry `n` of `CasualQueueDelayLevels` | Every casual matchmaking entry waits entry `n` of `CasualQueueDelaySeconds` before the search starts, for the next `CasualQueueDelayMatches` completed casual matches. The highest step reached applies. Ranked matchmaking and custom rooms are not delayed |

| Who | Rewards |
|---|---|
| Offender | No coins, pass XP, mission or medal progress from that match. AFK, returned and Leave, returned: rewards are kept |
| Partner of the offender (2v2) and opponents | Normal. The bot plays the offender's seat (R-50 to R-52); nobody else is penalised |

## 7. Record, notice and appeal

- Every offense is stored with the match id, the queue (ranked or casual), the seat, the timeout, reconnect and leave events with server timestamps, and the config version used.
- The player sees each active offense, its expiry, their current step on each ladder and how to leave it.
- An admin can remove an offense or lift a penalty, with a mandatory reason. Every admin change is audited.
- The matchmaking ban of step 3 is shown to the player with the ban reason category `afk_leaving` (owner decision 2026-11-22 on #186, `docs/admin.md` 6.2): the `MATCHMAKING_BAN` refusal carries `ban_reason` `AFK_LEAVING` with its end, never the offense evidence. Each ranked ladder ban is also recorded as a `penalty_ban` row of `account_sanctions` (category `afk_leaving`, its end, none while under review), written in the same transaction as the record; the record keeps its id as `BanSanction`, so the refusal can name it. Lifting the penalty from the panel, or by an upheld appeal, lifts that row.
- Appeal (owner decisions 4 and 5 of 2026-11-22 on #186): the `MATCHMAKING_BAN` refusal carries `appeal`, an `AppealGrant` with the sanction id and a short-lived appeal-only token (`docs/admin.md` 6.4, Appeals). The player sends `POST /appeals` with that token, the sanction id and a short explanation. One appeal per ban; a rejected appeal is not sent again for the same ban. Support lifts the ban in the appeal queue, which also clears the ban and the queue delays from the penalty record (like the lift action); the offenses stay. The queue delay alone (step 2) and warnings are not sanctions and have no appeal.
- Applying penalties is idempotent: replaying the same match end never records a second offense. The record keeps the match ids it has applied inside the offense window, so a replayed match end is answered with the current state and changes nothing.
- Implemented in `server/internal/penalties` (#478): one `penalty_records` row per account holds the `Record` as JSON. The hosting pod applies `MatchEnd` for every player of a matchmade match that ended with a result, under a row lock. The matchmaking gate refuses an entry with `MATCHMAKING_BAN` (with the ban end, or none while under review) and moves the search start by the queue delay of that queue's ladder. The values come from the `penalties` config domain (section 8); a match uses the version that was active when its room opened, and each offense keeps that version. The same holds for the rating: the room writes the `OffenseRatingPenalty` and `PartnerLossReductionPercent` of that version into its `matchstore.Outcome`, and the rating relay applies them, so a config change between the room opening and the relay changes nothing for that match. `GET /penalties` returns the player view for both ladders: level, step, the level of the next step, queue delay and matches left, ban end, review flag, clean streak against `CleanMatchesToForgive`, and each active offense with its kind, weight and expiry. The client reads it after a match to show the notice. Support lifts a penalty or removes one offense with a reason through `POST /admin/players/{id}/penalties/lift` and `.../remove` (`docs/admin.md`); both are audited.

## 8. Config

| Field | Meaning |
|---|---|
| `ReconnectGraceSeconds` | How long a player who quit or disconnected may stay away before it counts as a leave |
| `AfkReturnedWeight`, `AfkWeight`, `LeaveReturnedWeight`, `LeaveWeight` | Weight of each offense |
| `OffenseWindowDays` | How long an offense stays active |
| `CleanMatchesToForgive` | Clean completed matches in a row that remove the oldest active offense |
| `WarningLevel`, `QueueDelayLevel`, `BanLevel`, `ReviewLevel` | Offense level at which each step starts |
| `QueueDelaySeconds`, `QueueDelayMatches` | Queue delay length and how many matches it lasts |
| `BanHours`, `BanEscalationFactor` | First ban length and its growth for each further ban in the window |
| `OffenseRatingPenalty` | Extra rating loss for the offender |
| `PartnerLossReductionPercent` | How much smaller the partner's rating loss is |
| `CasualOffenseWindowDays` | How long a casual offense stays active |
| `CasualCleanMatchesToForgive` | Clean completed casual matches in a row that remove the oldest active casual offense |
| `CasualWarningLevel` | Casual offense level at which the casual warning starts |
| `CasualQueueDelayLevels`, `CasualQueueDelaySeconds` | Ascending casual offense levels of the casual queue delay steps, and the delay of each step; equal length. No entry of `CasualQueueDelaySeconds` exceeds `QueueDelaySeconds`, so the casual ladder stays lighter than the ranked one |
| `CasualQueueDelayMatches` | How many completed casual matches a casual queue delay lasts |

Values are set in the admin panel before launch (`/admin/config/penalties`, keys in `lowerCamelCase`, a key left out keeps its default). A match always uses the config version that was active when it started. Until the first publish the server runs on the placeholders of `penalties.DefaultConfig` (`server/internal/penalties`), which are not design decisions.

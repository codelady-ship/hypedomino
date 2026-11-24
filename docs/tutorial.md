# Tutorial

The interactive tutorial teaches a new player the rules of `docs/rules.md` in one short scripted hand against a bot (#88, Epic #77). It covers, in this order, the open ends (R-15, R-16, R-26), board points on a multiple of 5 (R-25, R-28, R-29), the spinner (R-17, R-18, R-27) and the target score of 365 (R-43).

## 1. Scripted hand

- **TU-1** The tutorial is one fixed 1v1 first hand. The player holds 2-3, 2-4, 3-5, 6-6, 0-4, 1-5 and 2-6; the bot holds 7 tiles, of which only the two it places are shown. Every move is scripted, so every player sees the same board and the same points.
- **TU-2** The steps, in order. A coach step only shows its text and moves on with Continue. A player step waits for the scripted tile on the scripted end. A bot step shows the bot's placement and its points at once, and moves on with Continue.

| # | Actor | Topic | Move | Open ends after | Points |
|---|---|---|---|---|---|
| 1 | Coach | Ends | `tutorial.welcome` | | |
| 2 | Player | Ends | 2-3 opens (R-9) | 2, 3 | 0, an opening scores only as R-29 allows |
| 3 | Coach | Ends | `tutorial.ends` | | |
| 4 | Bot | Ends | 3-3 on the 3 | 2, 3-3 = 8 | 0 |
| 5 | Player | Multiples of 5 | 2-4 on the 2 | 4, 3-3 = 10 | Player +10 |
| 6 | Coach | Multiples of 5 | `tutorial.multiples` | | |
| 7 | Bot | Spinner | 3-1 on the other side of 3-3, which becomes the spinner; its empty arms add nothing | 4, 1 = 5 | Bot +5 |
| 8 | Coach | Spinner | `tutorial.spinner` | | |
| 9 | Player | Spinner | 3-5 on the spinner's top arm | 4, 1, 5 = 10 | Player +10 |
| 10 | Coach | 365 | `tutorial.target` | | |
| 11 | Coach | 365 | `tutorial.done` | | |

- **TU-3** A player move with another tile is refused with `tutorial.wrong_tile`, and one on another end with `tutorial.wrong_end`. Nothing is placed and the step stays.
- **TU-4** The script is display data. The tutorial sends no intent, starts no match and changes no score, rating, mission, pass or statistic. The points it shows exist only on that screen. `TutorialScriptTests` replay the script through the rules of R-15 to R-29, so a script that breaks a rule or shows points the rules would not give fails the build.

## 2. First offer, skip and replay

- **TU-5** The player can skip the tutorial before it starts and at any step. A skipped run shows nothing further.
- **TU-6** A finished run, skipped or completed, can be played again from the first step with no points.
- **TU-7** A device that has finished a run, skipped or completed, keeps `tutorial.finished` (`docs/anti-cheat.md`, Device storage) and is no longer offered the tutorial on its own; the player can still open it again.
- **TU-9** The tutorial is offered on its own only in the Lobby: the first time in an app session that the app enters the Lobby and the device has no `tutorial.finished`, a prompt asks Play or Skip (`tutorial.offer.*`). A resumed match is not interrupted; the prompt waits for the Lobby after it. Play starts the run of TU-2 from the first step. Skip counts as a skipped run: the device keeps `tutorial.finished` (TU-7) and is never offered the tutorial on its own again.
- **TU-10** The prompt is shown at most once per app session. Leaving the Lobby before answering withdraws it without keeping `tutorial.finished`, so the next app session asks again. Replay (TU-6) is opened from a menu, never by this prompt.

## 3. Completion reward

- **TU-8** The tutorial grants no reward: completing, skipping or replaying it gives no currency, item, XP or progress, on any account or device. There is no tutorial ledger source in `docs/economy.md` and no server change for it (owner decision, #852).

## 4. Implementation

- Client: `Game/Tutorial` (`client/README.md`, Tutorial). `TutorialOffer` follows the app flow and answers TU-9 and TU-10; `TutorialBoot` builds it on the app flow.
- Not built yet: the tutorial screen and the prompt view with their art (#848), and the menu entry for replay (TU-6).

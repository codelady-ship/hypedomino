# Store listing and submission

What the Google Play and App Store pages say and show, the rules every word and image on them follows, the age rating and declaration answers, how the pages are tested after launch, and the checklist that has to be green before a build is submitted (#194, Epic 17 #189).

**Status: draft, awaiting owner approval.** The owner approves it by merging; the open decisions are section 10 (OQ-1 to OQ-8), each with a proposed default that applies until the owner decides otherwise. No art exists yet, the developer accounts are not opened and the legal entity is not fixed (`docs/privacy.md` OQ-11), so this document makes the submission ready to assemble, not ready to send.

Owned elsewhere and only referenced here: game rules (`docs/rules.md`), economy and purchase products (`docs/economy.md`), launch languages, text rules and the cultural checklist (`docs/localization.md` sections 1, 4 and 6), age rating targets (`docs/localization.md` section 7), Data safety and App Privacy answers (`docs/privacy.md` section 10), art rules (`docs/art-direction.md`), motion and effect levels (`docs/ui-motion.md`), device tiers (`docs/performance.md`), support and FAQ pages (`docs/launch-runbook.md` section 9), release gate (`docs/roadmap.md` section 4). Store limits and console fields are quoted as published at the time of writing; ST-6 and ST-10 check them again against the consoles at submission.

## 1. Inputs already decided

| Input | Value | Source |
|---|---|---|
| Distribution | Every store country except Russia. The soft launch subset is #196 | `docs/privacy.md` 9.3, OQ-2 |
| Listing languages | `az`, `en`, `tr`, `ru`; the Russian copy serves Russian-speaking players outside Russia | `docs/localization.md` 1 |
| Accounts | 13 and older only; neutral age gate | `docs/privacy.md` 7.3, 7.4 |
| Content rating target | IARC lowest tier, PEGI 3, App Store 4+ | `docs/localization.md` 7 |
| Payments | Store billing only; crystal packs, pass products and offers; no paid random items (chests are bought with earned coins only); coins and crystals have no cash value | `docs/economy.md` EP-3, EP-5, EP-7, section 7 |
| Communication | Emoji and preset quick messages only; no free text chat, no voice | `docs/localization.md` 5.7, #149 |
| Ads and tracking | None; no advertising id | `docs/privacy.md` P-3 |
| Support URL | The FAQ web pages | `docs/launch-runbook.md` 9.4 |

## 2. Fields

### 2.1 Copy fields

The text of every copy field lives in one file per locale, [`docs/store/listing.<locale>.md`](store/): `az`, `en`, `tr`, `ru`. Each field is a `## <field id>` heading followed by one fenced `text` block, pasted into the console exactly as written. Limits count characters (Unicode code points), newlines included.

| Id | Field id | Google Play | App Store | Limit |
|---|---|---|---|---|
| SF-1 | `play.title` | App name | | 30 |
| SF-2 | `play.short` | Short description | | 80 |
| SF-3 | `apple.name` | | Name | 30 |
| SF-4 | `apple.subtitle` | | Subtitle | 30 |
| SF-5 | `apple.keywords` | | Keywords, comma separated | 100 |
| SF-6 | `apple.promo` | | Promotional text (changeable without review) | 170 |
| SF-7 | `description` | Full description | Description | 4000 |
| SF-8 | `caption.S-1` to `caption.S-8` | Screenshot caption (section 5.3) | Screenshot caption | 32 |

- One description serves both stores, so a fact is changed once.
- SF-8 is this document's layout budget for caption overlays, not a store limit.
- An experiment variant of a field is the same heading with `@` and a capital letter (`play.short@B`), checked against the same limit (section 8).

### 2.2 Other listing fields

| Id | Field | Value |
|---|---|---|
| SF-9 | Developer or seller name (both stores) | `<publisher>`, the legal entity (`docs/privacy.md` OQ-11) |
| SF-10 | Contact email (both stores) | `<support-email>` (`docs/launch-runbook.md` 9.1, #144) |
| SF-11 | Website (Play), support URL (App Store) | `<faq-url>`, the FAQ pages (`docs/launch-runbook.md` 9.4) |
| SF-12 | Privacy policy URL (both stores) | `<privacy-policy-url>` (#195, `docs/privacy.md` L-27) |
| SF-13 | Account deletion web URL (Play) | `<account-deletion-url>` (#63, `docs/privacy.md` 10.3) |
| SF-14 | Marketing URL (App Store, optional) | Not set at launch |
| SF-15 | Phone (Play, optional) | Not set, unless EU trader status needs it (ST-2) |
| SF-16 | Category | OQ-2 |
| SF-17 | Tags (Play, up to 5) | Chosen from the console's list, in this order of preference: board, dominoes if offered, multiplayer, competitive multiplayer, casual. Only tags that describe the game |
| SF-18 | Default language | `en-US` on both stores (OQ-5) |

Placeholders in angle brackets are replaced before submission (ST-1, ST-7, ST-8). They never appear in the copy files; the check in section 4 rejects them there.

### 2.3 Locales

| Listing file | Google Play | App Store |
|---|---|---|
| `listing.en.md` | `en-US` (default) | English (U.S.) (primary) |
| `listing.az.md` | `az-AZ` | Azerbaijani if App Store Connect offers it; otherwise not used on the App Store (OQ-5) |
| `listing.tr.md` | `tr-TR` | Turkish |
| `listing.ru.md` | `ru-RU` | Russian |

## 3. Copy rules

| Id | Rule |
|---|---|
| CR-1 | Every statement is true for the build being submitted and is backed by a row in 3.1. A claim that no longer holds is removed from all four files in the same PR that changes its source. |
| CR-2 | Only our own game and our own rules are described, in their own terms. No third-party game, studio, product or brand name anywhere, keywords included (CLAUDE.md, `docs/localization.md` 6). |
| CR-3 | No price, discount or currency amount in any field. Prices come from the stores (`docs/economy.md` 7.4) and change without a copy change. |
| CR-4 | No gambling cues or real-money claims: no casino, betting, wager, jackpot, slot or roulette words, no "win money" or "real money" (`docs/localization.md` 6, `docs/economy.md` EP-4, EP-5). |
| CR-5 | No ranking, price or promotion words in the title, name, subtitle, short description, keywords or captions ("free", "best", "top", "#1", "new", "sale", "download"), no emoji there, no calls to install. The description may say "free" only for the free pass track, and currently does not. |
| CR-6 | The brand name is written `HypeDomino` in every locale, in Latin script (OQ-1). |
| CR-7 | Keywords: comma separated with no spaces around commas, no repeats, and no word already in the name or subtitle, since the App Store indexes those already. Every keyword describes the game truthfully. |
| CR-8 | The currency, the pass, ranks and game terms use the same words as the approved in-game text of that language (OQ-7). Until those keys exist, the files use "sikkə", "coin", "монеты" for coins. The pass is "Hype Pass" in every language (`docs/pass-missions-medals.md` OQ-11). |
| CR-9 | Each language reads as written by a native speaker for players of that market, not as a translation: `az` and `tr` are written separately, not converted from each other; `ru` uses common domino terms (дубль, базар, кости). A native reviewer approves each file (`docs/localization.md` 6.1 step 2); the owner approves the store-facing set (step 4). |
| CR-10 | Casing in `az` and `tr` uses the language's own capitals: `i`/`İ` and `ı`/`I` (`docs/localization.md` 4.6). Headlines are written in capitals in the file, never uppercased by a tool. |
| CR-11 | The copy is written for players aged 13 and older: competitive play, seasons and friends. It does not address children (`docs/privacy.md` 7.1, AR-14). |
| CR-12 | Rules are summarised, never contradicted. A simplification names the special cases it leaves out and points to the game's help pages. |

### 3.1 Claims

| Id | Claim in the copy | Source |
|---|---|---|
| CL-1 | Real-time online play; an internet connection is required | ADR 0003, ADR 0004 |
| CL-2 | 1v1 and 2v2, seven tiles each; 2v2 deals every tile, 1v1 draws from the boneyard when no tile can be placed | R-2, R-3, R-5, R-22 |
| CL-3 | In 2v2 partners sit opposite each other | R-6 |
| CL-4 | 1v1 and 2v2 are both ranked, each with its own rank and leaderboard, by season | R-53, `docs/ranking.md` 1, 7 |
| CL-5 | Casual play with the same rules and no rating | R-56 |
| CL-6 | Custom rooms with friends by invite link or room code; the creator sets up the teams; friend lobby between matches | R-55, `docs/friend-lobby.md` 1 to 3 |
| CL-7 | A placement whose open ends total a multiple of five scores that total; special cases named | R-25, R-28 to R-30 |
| CL-8 | The first double with tiles on both sides becomes the spinner and opens up to four ends; there is one per hand | R-17, R-18 |
| CL-9 | The team that plays all its tiles first scores the pips left in the opponents' hands; the 300 rule is named as a special case | R-32, R-33, R-41 |
| CL-10 | Matches are played to 365 points | R-43 |
| CL-11 | Tiles are shuffled on the server; nobody sees a tile they do not hold | R-4, NT-4 |
| CL-12 | The server checks every move, point and reward | NT-1, NT-2, ADR 0003 |
| CL-13 | Nothing for sale changes rank, tiles or rules; no pay-to-win | `docs/economy.md` EP-2 |
| CL-14 | A bot plays for a player who drops out until they return; the match goes on | R-51, R-52 |
| CL-15 | Cosmetic types: avatars, frames, tables, tile skins, emoji | `docs/cosmetics.md` 2 |
| CL-16 | Every shop offer shows exactly what it grants; a chest shows its pool, odds and pity before purchase | `docs/economy.md` EP-3 |
| CL-17 | Emoji and quick messages in a match; friends, online presence, invites | `docs/localization.md` 5.7, #151, #152, #153 |
| CL-18 | Coins are earned by playing; daily and weekly missions; a season pass; medals | `docs/economy.md` 2, `docs/pass-missions-medals.md` 1 to 5 |
| CL-19 | No ads | `docs/privacy.md` P-3 |
| CL-20 | Four languages | `docs/localization.md` 1 |
| CL-21 | Accounts 13 and older | `docs/privacy.md` 7.3 |
| CL-22 | Optional purchases: crystal packs, the premium pass, offers; coins and crystals have no cash value and cannot be withdrawn | `docs/economy.md` EP-5, 7.1 to 7.3 |
| CL-23 | The game's help explains the special cases | `docs/launch-runbook.md` 9.4 (FAQ "How to play") |

Claims deliberately left out until they are measured or built: frame rate and device performance (`docs/performance.md` gates are unmeasured), offline or practice play (none exists), any tournament or event format (#146 has not defined one).

## 4. Check

`node tools/store-listing/store-listing.mjs [directory]` checks every listing file (default `docs/store`); CI runs its tests through `npm run test:tools`, which also run it on the committed files.

| Check | Rule |
|---|---|
| Fields | Every field of 2.1 present once in every locale; no unknown field; every heading has a text block |
| Limits | Each value within its limit in 2.1 |
| Forbidden terms | CR-3 and CR-4 term list in all four languages, applied to every locale and every field, whole words, case folded with the Turkish `İ`, `I`, `ı` mapped to `i` (`docs/localization.md` 5.2 step 3) |
| Promotional terms | CR-5 list in all four languages, in every field except the description and promotional text |
| Currency | No currency symbol or currency name |
| Emoji, placeholders | None in any field |
| Keywords | CR-7 |

The term lists are generic on purpose: they name no game or product (CR-2). What a tool cannot check, truth (CR-1), native quality (CR-9) and the audience (CR-11), is checked by the reviewers in ST-4 and ST-5.

## 5. Graphics

No art exists yet. Production belongs to Epic 15 (#165): UI kit #168, tiles and tables #169, avatars #171 and #172, frames #173, table backgrounds #174, emoji #175, effects #176, sound #177; the style is `docs/art-direction.md` (#166) and the workflow `docs/asset-workflow.md` (#167). Every store graphic passes the cultural review (`docs/localization.md` 6.1 step 4).

### 5.1 Icon

| Id | Requirement |
|---|---|
| IC-1 | One icon design for both stores and every locale. Original, no text or letters, no coins, chips, cards or other casino cues, no rarity marks (`docs/art-direction.md` 2, 3, 8). Reads as a silhouette at small sizes |
| IC-2 | Colours follow `docs/art-direction.md` section 2: named tokens for UI and role colours, cosmetic art in its own reviewed palette. No new colour is defined here |
| IC-3 | Google Play listing icon: 512 x 512 px, 32-bit PNG, at most 1024 KB, full square. Play applies the mask and shadow, so the file has neither |
| IC-4 | Android launcher: adaptive icon with a foreground and a background layer of 108 x 108 dp, the motif inside the central 66 dp safe zone (72 dp visible), plus a monochrome layer for themed icons. Legacy and round icons are generated from it in Unity Player Settings |
| IC-5 | App Store icon: 1024 x 1024 px PNG, no transparency, square corners (iOS applies the mask). Dark and tinted variants are supplied for the iOS home screen appearances |
| IC-6 | A changed icon in a test (section 8.2) must be included as an alternate icon in the iOS binary before the test can use it |

### 5.2 Screenshot sizes and counts

Orientation follows OQ-3 (default landscape); only that orientation is produced. Device classes follow OQ-4 (default: Android phones and tablets, iPhone only).

| Id | Store and device | Size per image | Count | Format |
|---|---|---|---|---|
| SS-1 | Google Play phone | 16:9 or 9:16, 1920 x 1080 (landscape) or 1080 x 1920 (portrait); Play accepts 320 to 3840 px with the long side at most twice the short side, and asks for at least 4 images of 1080 px or more for recommendation eligibility | 8 (Play allows 2 to 8) | JPEG or 24-bit PNG, no alpha |
| SS-2 | Google Play 7-inch tablet | 16:9 or 9:16, short side 1080 to 7680 px | 8 | As SS-1 |
| SS-3 | Google Play 10-inch tablet | As SS-2 | 8 | As SS-1 |
| SS-4 | Google Play feature graphic | 1024 x 500 | 1 per locale | JPEG or 24-bit PNG, no alpha |
| SS-5 | App Store iPhone 6.9-inch | 2868 x 1320 (landscape) or 1320 x 2868 (portrait); smaller iPhone sizes are scaled from it | 8 (App Store allows 1 to 10) | PNG or JPEG, RGB, flattened, no transparency |
| SS-6 | App Store iPad 13-inch | 2752 x 2064 or 2064 x 2752; only if OQ-4 adds iPad | 8 | As SS-5 |

- Every image set exists once per listing locale (2.3): the UI in the capture is in that locale's language and the caption comes from that locale's file.
- The first three images carry the decision (search results show them first): S-1, S-2 and S-3 in that order.
- The feature graphic carries no words other than the `HypeDomino` name (CR-6); it shows the table and cosmetics, never a price or a call to install.

### 5.3 Shot list

| Shot | Screen | State to capture | Caption field |
|---|---|---|---|
| S-1 | 2v2 match | A placement that scores: open ends on a multiple of five, `scoreCascade` at its peak step, both team meters visible | `caption.S-1` |
| S-2 | Friend lobby (FL-1) | Four members as their avatars in animated frames, one emoji above an avatar | `caption.S-2` |
| S-3 | Rank screen | Ranked 1v1 and 2v2 ranks side by side or the mode switch visible, a leaderboard behind | `caption.S-3` |
| S-4 | 2v2 match | A spinner with tiles on all four arms | `caption.S-4` |
| S-5 | Team setup (FL-2) | Teams split, the invite panel open with the link and room code | `caption.S-5` |
| S-6 | Collection | Avatars and tables of several rarity tiers, rarity drawn by the client | `caption.S-6` |
| S-7 | Pass | Both tracks with exact rewards shown, a mission panel | `caption.S-7` |
| S-8 | Match end | The `domino` banner with emoji reactions from the seats | `caption.S-8` |

### 5.4 Capture rules

| Id | Rule |
|---|---|
| CP-1 | Every frame is captured from a real build: the release candidate or a build from the same commit, on the reference devices of `docs/performance.md` (high tier for the stills). No mock-up, render or composited gameplay. The caption band and an optional device frame are the only additions |
| CP-2 | Captures run against staging with staff test accounts. No real player's nickname, avatar or data appears (`docs/privacy.md` 5); test nicknames pass the nickname policy (`docs/localization.md` 5) |
| CP-3 | Only items that are `live` in the launch catalogue appear (`docs/launch-content-gate.md` 2); nothing that a player cannot get at launch |
| CP-4 | No real-money price string is visible. A coin price on a shop card may show, since it is game currency |
| CP-5 | Effects at the high effect level as shipped, no added effects; flashing within the photosensitivity limits (`docs/localization.md` 6) |
| CP-6 | Clean status bar, no debug overlay, no `[!key]` missing-text marker (`docs/localization.md` 2.3) |
| CP-7 | Captions are an overlay drawn per locale from the listing file, outside the captured frame; they are store graphics, not game art, so the no-text-in-art rule of `docs/localization.md` 6 still holds for the game |

### 5.5 Trailer and app previews

One 30-second master edit serves both stores: the Google Play promo video (a YouTube link) and the App Store app preview (15 to 30 seconds per Apple). Beat timings are a proposal for the editor.

| Id | Seconds | Shows |
|---|---|---|
| TR-1 | 0 to 3 | The hook: a scoring placement and `scoreCascade`, no logo first |
| TR-2 | 3 to 9 | A 2v2 hand: partners across the table, a double turning into the spinner, a second scoring placement |
| TR-3 | 9 to 14 | A 1v1 hand: a draw from the boneyard, the last tile, the `domino` banner |
| TR-4 | 14 to 19 | The friend lobby: avatars in frames, emoji |
| TR-5 | 19 to 24 | Ranked: a rank up (`rankUp`), a leaderboard |
| TR-6 | 24 to 30 | The collection, then an end card with the icon and `HypeDomino` |

| Id | Requirement |
|---|---|
| TV-1 | Footage only from real builds under the capture rules (CP-1 to CP-6); no hands, devices or people on screen |
| TV-2 | Audio: game music and sound effects (#177), no voice-over. Captions per locale from caption keys of the shot list, because previews autoplay muted |
| TV-3 | App Store: one preview per locale and device class of 5.2, at the preview resolution App Store Connect lists for that class (iPhone 6.9-inch: 1920 x 886 landscape or 886 x 1920 portrait), H.264 or ProRes 422 HQ, up to 30 fps, at most 500 MB; a poster frame from TR-1 or TR-2 |
| TV-4 | Google Play: 1920 x 1080 (or the chosen orientation), uploaded to YouTube as public or unlisted, embeddable, monetisation off, not age restricted; the link to the video itself, not a playlist or channel |
| TV-5 | No price, no install call, no ranking claim (CR-3, CR-5) |

## 6. Age rating

The answers follow from the documents named per row. The questionnaire wording changes over time, so answers are mapped to the console's current questions at submission (ST-12, ST-13); a question not covered here is answered from the same sources and added to this table in a PR.

### 6.1 IARC (Google Play and the other IARC stores)

| Id | Topic | Answer | Source |
|---|---|---|---|
| AR-1 | Category | Game | |
| AR-2 | Violence, blood, gore | No. Cartoon impact effects only, no weapons, no harm to anyone shown | `docs/localization.md` 6, `docs/art-direction.md` 8 |
| AR-3 | Fear, horror | No | `docs/localization.md` 6 |
| AR-4 | Sexuality, nudity, suggestive themes | No | `docs/localization.md` 6 |
| AR-5 | Language: profanity, crude humour, discrimination | No. All text is ours and reviewed; player names are filtered and moderated | `docs/localization.md` 5.3, 6 |
| AR-6 | Alcohol, tobacco, drugs | No | `docs/localization.md` 6 |
| AR-7 | Simulated gambling | No. Dominoes scoring is not a wager; no casino presentation | `docs/economy.md` EP-4, `docs/localization.md` 6 |
| AR-8 | Real-money gambling, or winning money or items of real value | No. No entry fees, no stakes, no transfers; coins have no cash value and cannot be withdrawn | `docs/economy.md` EP-4, EP-5 |
| AR-9 | Digital purchases | Yes: crystal packs, pass products, offers | `docs/economy.md` 7 |
| AR-10 | Purchasable random items | No. Chests have random contents but are bought only with coins, which are earned by play and never sold, directly or through crystals | `docs/economy.md` EP-3 |
| AR-11 | Users interact or communicate | Yes. Emoji and preset quick messages from a fixed catalogue, friend invites, nicknames and custom room names that other players see. No free text chat, no voice, no image or file sharing. Names are filtered, players can report, block and mute | `docs/localization.md` 5.7, 5.8, #149, #155 |
| AR-12 | Shares the user's current physical location with other users | No. The account country is visible through country leaderboards, but it is a profile setting, not a current location | `docs/privacy.md` D-8, 10.3 |
| AR-13 | Unrestricted internet access | No. The in-game help shows only our own FAQ pages (`docs/launch-runbook.md` 9.4); this holds only if that view cannot browse outside them (ST-16) | |
| AR-14 | Personal information shared with third parties | No | `docs/privacy.md` 10.1 |

Expected result: the lowest age tier, with the "Users Interact" and "In-Game Purchases" notices. The questionnaire, not this table, decides the rating.

### 6.2 App Store age rating

| Id | Topic | Answer | Source |
|---|---|---|---|
| AA-1 | Violence (cartoon or fantasy, realistic, prolonged or graphic) | None | AR-2 |
| AA-2 | Horror or fear themes | None | AR-3 |
| AA-3 | Sexual content, nudity, mature or suggestive themes | None | AR-4 |
| AA-4 | Profanity or crude humour | None | AR-5 |
| AA-5 | Alcohol, tobacco or drug use or references | None | AR-6 |
| AA-6 | Medical, treatment or wellness information | None | |
| AA-7 | Simulated gambling | None | AR-7 |
| AA-8 | Gambling with real money, contests with prizes of real value | No | AR-8 |
| AA-9 | Loot boxes or other paid chance-based items | No | AR-10 |
| AA-10 | Unrestricted web access | No, under the AR-13 condition | AR-13 |
| AA-11 | User-generated content | Yes: nicknames and custom room names, filtered and moderated, with report and block | AR-11 |
| AA-12 | Messaging or chat | Yes, limited to emoji and preset quick messages; answered as the console defines the feature (OQ-6) | AR-11 |
| AA-13 | Advertising | No | `docs/privacy.md` P-3 |
| AA-14 | Parental controls | No | |
| AA-15 | Age assurance | Self-declared age gate only, unless #73 adds a store age signal (`docs/privacy.md` 7.4); the answer follows what #73 ships | |
| AA-16 | Made for Kids | No | `docs/privacy.md` 7.3 |

Target: 4+ (`docs/localization.md` 7). If App Store Connect computes a higher rating from AA-11 and AA-12, OQ-6 applies.

### 6.3 Audience declarations

| Id | Store | Answer | Source |
|---|---|---|---|
| AU-1 | Google Play target age groups | 13 to 15, 16 to 17, 18 and over. Not designed for children; not in the Families programme | `docs/privacy.md` 7.3 |
| AU-2 | Google Play "could the listing unintentionally appeal to children" | No (OQ-8): the copy addresses competitive teen and adult players (CR-11), the age gate is neutral | `docs/privacy.md` 7.1, 7.4 |
| AU-3 | Turkey and other local systems | The local rating shown by each store is checked at submission | `docs/localization.md` 7 |

## 7. Data and other declarations

Google Play Data safety and Apple App Privacy are answered from `docs/privacy.md` 10.1 and 10.2 exactly (L-24, L-25); they are not repeated here. The remaining console declarations:

| Id | Store | Declaration | Answer | Source |
|---|---|---|---|---|
| DC-1 | Play | Contains ads | No | `docs/privacy.md` P-3 |
| DC-2 | Play | Advertising ID | Not used; `AD_ID` removed from the manifest | `docs/privacy.md` 10.3, #76 |
| DC-3 | Both | App access for review | Every feature needs sign-in with Google or Apple; the reviewer signs in with their own account. A lone reviewer finishes a casual match because casual queues seat bots after `CasualBotFillSeconds` (`docs/matchmaking.md` 2), so that value is above 0 in production during review. Custom rooms and the friend lobby need a second account: the review notes say so and give the path | R-56, `docs/friend-lobby.md` |
| DC-4 | Play | Government, financial, health, news app | No | |
| DC-5 | Apple | Sign-in options | Sign in with Apple is offered next to Google | #56, #57 |
| DC-6 | Apple | Account deletion in the app | Yes | `docs/privacy.md` 8.2, #63 |
| DC-7 | Both | Digital goods through store billing only | Yes | `docs/economy.md` EP-7 |
| DC-8 | Apple | Export compliance | Standard encryption only (TLS for the transport and downloads), which is exempt; `ITSAppUsesNonExemptEncryption` false, confirmed against ADR 0004 at submission | ADR 0004 |
| DC-9 | Both | Third-party content rights; AI-generated content | All art is ours or generated under recorded terms with provenance; the owner confirms after the AI art terms review. AI use is disclosed only where a store requires it | `docs/asset-workflow.md` 3, O-2 to O-4 |
| DC-10 | Both | EU trader status (Digital Services Act) | Trader, with the entity's address, phone and email shown on the listing as the stores require; waits on the legal entity | `docs/privacy.md` OQ-11 |

## 8. Store testing

### 8.1 Rules

| Id | Rule |
|---|---|
| TS-1 | Tests start only after the soft launch (#196) brings enough store traffic for the consoles to reach a result; the first submission ships the default page |
| TS-2 | One variable per test. Copy variants go into the listing files as `@B`, `@C` fields and pass the same check and review (section 4, CR-9); graphic variants pass the same capture rules and cultural review |
| TS-3 | Google Play decides on retained first-time installers at the console's 90% confidence setting. App Store decides on conversion rate as reported by product page optimization |
| TS-4 | A test runs for at least one full week so weekday and weekend traffic are both in it, and stops when the console reports a result or the test limit is reached |
| TS-5 | The winner is applied to all locales where it makes sense, the losing variant fields are deleted, and the result is recorded in #194 or its follow-up |

### 8.2 Mechanisms

| Store | Mechanism | Limits that shape the plan |
|---|---|---|
| Google Play | Store listing experiments: a default graphics experiment for the default listing, localized experiments per language | One default graphics experiment and up to five localized experiments at a time, each with up to three variants against the current page. Text (short and full description) is tested only in localized experiments |
| App Store | Product page optimization | One test at a time, up to three treatments against the original, up to 90 days, treatments reviewed by App Review. Tests icon, screenshots and previews, not text. A test icon must already be in the binary (IC-6) |

### 8.3 Backlog

| Id | Store | Variable | Variants | Why |
|---|---|---|---|---|
| EX-1 | Both | First screenshot | S-1 scoring moment against S-2 friend lobby first | Learns whether play or social sells the game |
| EX-2 | Play, `az` and `tr` localized | Short description | Rules hook against social hook (`play.short@B`) | Primary markets first |
| EX-3 | Both | Icon | Tile motif against avatar motif | Needs two approved icon designs (IC-1) and IC-6 |
| EX-4 | Play | Feature graphic | Table scene against a collection of avatars and frames | Default graphics experiment |
| EX-5 | Both | Video | With against without the promo video or preview | Measures the trailer before a second edit is made |

## 9. Pre-submission checklist

Release manager owns the list and records its evidence in the release issue; it is the evidence of go/no-go item GN-23 (`docs/launch-runbook.md` 3.2). Every item is checked, or waived by an owner comment on #194.

| Id | Item | Owner | Evidence |
|---|---|---|---|
| ST-1 | Legal entity fixed; Google Play and Apple developer accounts opened as that organisation; `<publisher>` replaced (SF-9) | Product owner | Account pages; `docs/privacy.md` L-1 |
| ST-2 | EU trader status declared in both consoles (DC-10) | Product owner | Console screenshots |
| ST-3 | OQ-1 to OQ-8 decided or their defaults confirmed; this document's status updated | Product owner | Merged PR |
| ST-4 | Copy approved by a native reviewer per language and by the owner; `node tools/store-listing/store-listing.mjs` passes on the commit the copy is taken from | Release manager | Review record, command output |
| ST-5 | Every claim in 3.1 re-checked against the current documents and the release build | Release manager, QA lead | Checked list in the release issue |
| ST-6 | Copy pasted into both consoles from the approved commit; console counters agree with the limits of 2.1; Azerbaijani availability in App Store Connect checked (OQ-5) | Release manager | Console screenshots |
| ST-7 | Privacy policy, FAQ and account deletion pages live in all four languages, URLs entered (SF-11 to SF-13) | Support lead, Product owner | Links; `docs/privacy.md` L-27 |
| ST-8 | Contact email live and monitored (SF-10) | Support lead | Test email answered |
| ST-9 | Icon final in every variant of 5.1 and approved in the cultural review | Owner | Manifest record, files |
| ST-10 | Screenshots and feature graphic for every locale and device class of 5.2, from the release candidate under 5.4; sizes checked against the consoles' current specs | Release manager | Upload screenshots |
| ST-11 | Trailer and previews per 5.5; YouTube link unlisted or public with monetisation off | Release manager | Links |
| ST-12 | IARC questionnaire answered from 6.1; certificate received; ratings match `docs/localization.md` 7 or the owner accepted the difference | Release manager | Certificate |
| ST-13 | App Store age rating answered from 6.2; rating 4+ or OQ-6 applied | Release manager | Console screenshot |
| ST-14 | Audience declarations from 6.3 | Release manager | Console screenshot |
| ST-15 | Data safety and App Privacy from `docs/privacy.md` 10.1 and 10.2 | Product owner | `docs/privacy.md` L-24, L-25 |
| ST-16 | Declarations of section 7; the in-game help view cannot browse outside our FAQ pages (AR-13) | Release manager, Client lead | Console screenshots, device test |
| ST-17 | In-app products created with the ids of `docs/economy.md` 7 and matching the admin shop; country prices pushed from the published `store_prices` table (or set from its CSV export as the fallback) per `docs/economy.md` 7.4, Azerbaijan and Turkey among them; product names and descriptions localized in four languages | Live-ops lead | GN-22 evidence |
| ST-18 | Category and tags set (SF-16, SF-17) | Release manager | Console screenshot |
| ST-19 | Countries: every store country except Russia, or the soft launch subset of #196 | Release manager | Console screenshot; `docs/privacy.md` 9.3 |
| ST-20 | Review notes entered (DC-3); `CasualBotFillSeconds` above 0 in the production config | Release manager, Live-ops lead | Notes text, config version |
| ST-21 | Release held for manual release (App Store) and managed publishing (Play) | Release manager | `docs/launch-runbook.md` 4, T-7d |
| ST-22 | If the Play account is a personal account, the closed test that Play requires before production access is done (#193) | Release manager | Console status |
| ST-23 | The app name is available in both stores and clear of conflicting trademarks in the launch markets | Product owner | Search record |

## 10. Open questions

Each default applies until the owner decides otherwise on #194.

| Id | Question | Proposed default | Why |
|---|---|---|---|
| OQ-1 | App name per store; localize the name or not | `HypeDomino` plus a localized descriptor, the same on both stores: `HypeDomino: Online Dominoes`, `HypeDomino: Onlayn Domino`, `HypeDomino: Online Domino`, `HypeDomino: Домино онлайн`. The brand stays Latin in every locale | The descriptor carries the search term of each market; one brand spelling keeps the reserved-name check (`docs/localization.md` 5.3) and the icon end card simple |
| OQ-2 | Category: Board or Casual | Google Play: Board. App Store: Games, subcategories Board and Casual | The game is a board game by rules; Casual on the App Store adds reach without misdescribing it |
| OQ-3 | Screen orientation (open in `docs/ui-motion.md` 9) | Landscape | Four seats around a table and a chain growing in two directions use the long side; screenshots and previews are produced for the chosen orientation only |
| OQ-4 | Device classes | Android phones and tablets, with tablet screenshots; iOS iPhone only, so no iPad screenshots or previews | The device matrix (`docs/performance.md`) has no tablet tier yet; Play installs on tablets anyway, while an iPhone-only iOS build keeps iPad out of scope until a tablet tier is measured |
| OQ-5 | App Store primary language; Azerbaijani on the App Store | Primary English (U.S.). If App Store Connect offers no Azerbaijani metadata, the `az` file is used on Google Play only and Azerbaijani players on iOS see the English page | English is the final fallback (`docs/localization.md` 1.2) |
| OQ-6 | Quick messages and names as "Messaging" and "User-generated content" on the App Store, if they raise the rating above 4+ | Answer honestly and accept the computed rating; update `docs/localization.md` 7 | Changing the feature to keep a label is a product decision, not a listing one |
| OQ-7 | In-game names of the coin and the pass per language | Copy uses "sikkə", "coin", "монеты" until the approved text keys exist, then follows them (CR-8). The pass is "Hype Pass" in every language (owner decision 2026-11-12) | The store page and the game must use one word |
| OQ-8 | Answer to Play's "could the listing unintentionally appeal to children" | No, with the copy and graphics kept to CR-11 and the cultural checklist | Option A (`docs/privacy.md` 7.3) only holds if the listing does not target children |

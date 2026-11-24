# Sound

Status: proposal; the owner approves it by merging. The sound class of Epic 15 (#177): which sounds exist, when each one plays, file formats, loudness, folders and the license record. Groups and volume settings are `docs/ui-motion.md` section 6 (Sound); the content rules are `docs/localization.md` section 6 ("No sudden loud sounds"). The client plays them as section 6 describes.

## 1. Sounds

Every sound belongs to one group of `docs/ui-motion.md` (Sound). Match sounds follow server views like the haptics there: the first view of a connection plays nothing, so a reconnect replays no sound, and they never play on input alone. `ui_click` follows input, as it confirms a press of the interface.

| Name | Group | Plays when | Length |
|---|---|---|---|
| `tile_place_01` to `tile_place_04` | sfx | A placement settles on the board (`docs/ui-motion.md`, Tile place), any seat. One of the four, never the same one twice in a row | 0.30 s |
| `tile_draw` | sfx | A 1v1 player draws a tile from the boneyard (R-22) | 0.36 s |
| `pass` | sfx | A seat passes (R-21, R-23), with the pass label | 0.45 s |
| `turn_warning` | sfx | The local player's turn timer reaches `TurnWarningSeconds` (`docs/rules.md` section 11), once per turn | 0.12 s |
| `score` | sfx | The local team's score rose by less than the first score cascade step. With the default steps every rise reaches a step, so `combo_<step>` plays instead | 0.90 s |
| `combo_1` to `combo_5` | sfx | The local team's score rose: the step of the score cascade (`docs/vfx.md` section 5, `docs/ui-motion.md` section 8; chain boost included, as the VFX use it) picks `combo_<step>`, and step 6 (30 points or more) reuses `combo_5`. An ascending set of five steps on one scale. Owner decision 2026-11-16 (S-1) | 0.75 s |
| `hand_end` | sfx | A hand ends (domino or block, `docs/rules.md` sections 7 and 8), when the hand summary opens | 1.60 s |
| `match_win` | sfx | The results screen opens with the outcome won (`docs/ui-motion.md`, Match result) | 2.80 s |
| `match_lose` | sfx | The results screen opens with the outcome lost. A viewer without an outcome hears neither | 2.20 s |
| `ui_click` | ui | A button of the interface is pressed | 0.06 s |
| `lobby` | music | Loops while the lobby and the menus are shown | 72 s loop |
| `match` | music | Loops while a match is shown | 60 s loop |

The set holds no reward-reveal, chest or currency sound: such a sound must avoid casino cues (`docs/economy.md`, chest reveal; `docs/localization.md` section 6) and has no screen to belong to yet.

## 2. Folders

Two items under `assets-src/sound/` (`docs/asset-workflow.md` 4.1), one per script:

```text
assets-src/sound/
  sfx/                         sfx and ui groups
    provenance.md
    work/sfx.py                renders every file below
    work/sfx_<name>.wav        master
    export/<name>.ogg          compressed copy
  music/
    provenance.md
    work/music.py
    work/music_<name>.wav
    export/<name>.ogg
```

`python -I assets-src/sound/<item>/work/<item>.py assets-src/sound/<item>/work assets-src/sound/<item>/export` from the repository root writes all files of an item. The scripts are deterministic: two runs write byte-identical files on the same Python, NumPy, SciPy and libsndfile versions (recorded in each `provenance.md`). A changed sound is made by changing the script and running it again, never by editing a file by hand. WAV and Ogg files go through Git LFS (`.gitattributes`).

## 3. Formats

| File | Format |
|---|---|
| Master, `work/` | WAV, 48 kHz, 16-bit PCM, rounded without dither. Mono for sfx and ui, stereo for music |
| Compressed copy, `export/` | Ogg Vorbis written by libsndfile with libvorbis, compression level 0.6 of libsndfile's 0 to 1 scale, same channels and rate as the master. The Ogg stream serial is fixed and the page checksums rewritten, so the file is reproducible |

The Unity client imports the masters, not the Ogg files: Unity encodes every clip again for its target platform, so importing an Ogg would encode it twice. The Ogg files serve everything outside the Unity build (store trailer TV-2 of `docs/store-listing.md`, previews in the admin and on the web) and record the compressed size. MP3 is not used, because its encoder padding breaks a seamless loop.

## 4. Loudness

| Group | Target |
|---|---|
| music | Integrated loudness -16.0 LUFS (±0.05), true peak at most -1.0 dBTP |
| sfx, ui | Peak at a per-sound target of -1 to -6 dBTP (the table in `assets-src/sound/sfx/provenance.md`), lowered further when needed so that the highest momentary loudness stays at or under -14 LUFS |

- Loudness is ITU-R BS.1770-4 as the scripts implement it: K-weighting with the 48 kHz coefficients of the standard, 400 ms blocks with 75% overlap, an absolute gate at -70 LUFS and a relative gate at -10 LU; momentary loudness is the 400 ms window. A stereo 1 kHz sine at -23 dBFS measured -22.99 LUFS with the same code (EBU Tech 3341 expects -23.0).
- True peak is an approximation of BS.1770 annex 2: 4 times oversampling with SciPy's polyphase resampler, not the filter of the standard.
- The momentary ceiling keeps long chimes and jingles from standing out over the short clacks at the same peak, for the "No sudden loud sounds" rule. The per-group volumes of `docs/ui-motion.md` set the balance between groups in the game.
- A music loop is measured as a loop: the filters run over two copies and the second is measured, and the true peak wraps around the seam. A circular look-ahead limiter in `music.py` holds the true peak if a mix would exceed it.
- Decoding the Ogg copy moves true peaks by up to 0.9 dB in either direction; the highest is -0.9 dBTP and no decoded sample clips. The measured values are in each `provenance.md`.

## 5. Loops

A music loop is rendered as one period of a periodic signal: notes, reverb tails and the limiter wrap from the end of the loop to its start, so the last sample flows into the first. Tempos are chosen so that a beat is a whole number of samples (80 BPM is 36,000 samples, 96 BPM 30,000). The seam is checked at sample level in each `provenance.md`. In Unity the loop plays with `AudioSource.loop`.

## 6. Client import

The masters are copied into `client/Assets/HypeDomino/UI/Audio/Clips/Sfx/<name>.wav` and `Clips/Music/<name>.wav` (Git LFS) and imported by `HypeDomino.Editor.SoundContent` (menu HypeDomino > Import Sounds; batch `ImportAllAndExit` through the `unity-editor.yml` workflow), which also writes `UI/Audio/Resources/SoundBank.asset`, the clip of each sound and track. The import, one setting for every platform:

| Setting | sfx, ui | music |
|---|---|---|
| Source | `work/<item>_<name>.wav` | `work/music_<name>.wav` |
| Load Type | Decompress On Load | Streaming |
| Compression Format | ADPCM | Vorbis |
| Force To Mono | On | Off |
| Normalize | Off, so the mastered peaks of section 4 stay | Off |
| Preload Audio Data | On | Off |

Short clips decompressed at load cost no decode time during a match, which matters for the 0 B per frame and frame-time budgets (`docs/performance.md`); streaming keeps the two music loops out of memory. The Vorbis quality stays at Unity's default until the sizes in the build are measured on the device check (S-2).

### 6.1 Playback

- `UI/Audio/SoundPlayer` holds a fixed pool of 8 voices and 2 music sources, made once. A sound takes the next voice in turn at its group's level (`docs/ui-motion.md`, Sound); while its group is at 0 it is not started. A ninth sound inside the length of the first cuts the oldest voice.
- `tile_place` picks one of its four variants at random, never the last one again.
- Music loops with `AudioSource.loop`. A new track crossfades from the playing one over `motion.duration.slow` (480 ms). While the music group is at 0 no track plays; raising it starts the track of the current screen again. Login and lobby play `lobby`, a match plays `match`, the boot, update and maintenance screens play none.
- A volume change applies at once to the playing voices and the music. An app pause stops the voices, ends a crossfade and pauses the music, and nothing starts until the app resumes.
- Match sounds come from `Game/Match/SoundCues`, which reads each server view as the haptics do: the first view of a connection plays nothing. The client sees no pass or draw event, so it reads them from two views of the same hand: a draw is a smaller boneyard; a pass is a turn that moved to another seat with no new tile, no draw and no placement earlier in the turn (so ending a turn of several doubles, R-31, is no pass). The last placement of a hand arrives in the same view as the next deal, so it plays `hand_end`, not `tile_place`. `tile_place` plays when the view arrives; it moves to the settle of the tile (section 1) when the match screen drives `TilePlaceLayer`.
- A view that ends the match plays only `match_win` or `match_lose`, and a view that ends a hand plays only `hand_end`, so the short sounds of that view do not stack on the longer one ("No sudden loud sounds").
- A view in which the local team scores plays one scoring sound: `combo_<step>` from `EffectCues.StepOf`, the step the score cascade of `MatchEffects` shows, or `score` when the rise is below the first step. The combo replaces `score`, so the two never stack.
- `turn_warning` plays once per local turn when `TurnTimer.IsWarning` becomes true (`SoundCues.ReadTurnWarning`), from the timer tick of `Game/Match/MatchScreen`. It plays only while the current seat of the latest view is the viewer's own seat, so another seat's turn and a viewer without a seat hear nothing, and the timer follows server time, so it stays silent until the clock is synced.
- `ui_click` plays from `UI/Audio/ClickSound`, added to a button; a button that is not interactable is silent.

## 7. License record

| Field | Value for every sound of this class |
|---|---|
| artSource | `handmade`: no file came from a generator (`docs/asset-workflow.md` section 3). Every sample is computed by the item's script |
| Source material | None. No recording, sample, sample library, preset, sound font, loop, MIDI file or audio generator service was used, and no existing melody was transcribed |
| Authors | The scripts, the synthesis models and the compositions were written by an AI agent (Claude Code) on 2026-11-15. RC-5 (human work) applies as for scripted image work: none yet |
| Tools | Python, NumPy, SciPy, soundfile with libsndfile and libvorbis; the versions are in each `provenance.md`. These are tools, their output carries no third-party rights |
| Composition | Melodies are seeded random walks on a pentatonic scale over common chord progressions; the seeds are in `music.py` |
| Per sound | Script, function and seed in the item's `provenance.md` |

`provenance.md` of each item follows `docs/asset-workflow.md` section 3 where it applies to sound. Image-only checks (RC-3 reverse image search, RC-4) do not apply; a listening check for likeness to existing music is S-3.

## 8. Open items

| Id | Item | Owner |
|---|---|---|
| S-1 | Decided 2026-11-16 by the owner: `combo_1` to `combo_5` play on the score cascade steps (section 1) | Done |
| S-2 | Listening check by a person on Mid and High devices, speaker and headphones: levels between groups, "No sudden loud sounds", the loop seams after Unity's encoding, Vorbis quality and build size | Art team, with the wiring issue |
| S-3 | Listening check that no melody resembles an existing work | Owner or delegate, with the cultural review |
| S-4 | Self-check and cultural review of the sounds (`docs/localization.md` 6.1 steps 1 and 3) | Asset author, owner or delegate |
| S-5 | Client wiring: done (section 6.1), `turn_warning` included, except the placement timing at the settle: `tile_place` moves to the settle when the match screen drives `TilePlaceLayer` | Client, follow-up of #177 |

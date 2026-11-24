# sound/sfx

The sfx and ui sounds of `docs/sound.md` (#177): 17 short sounds synthesized by one script, `work/sfx.py`. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: handmade. No file came from a generator; every sample is computed by `work/sfx.py` (`docs/asset-workflow.md` section 3, `docs/sound.md` section 7)
- Authors: script, synthesis models and sound design by an AI agent (Claude Code) on 2026-11-15. No person has made or listened to these sounds
- Generator: none. No audio or image generator, no sample, recording, sample library, preset or sound font
- Terms: not applicable, no generator or third-party material was used. The tools are Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3, soundfile 0.13.1 with libsndfile 1.2.2 and libvorbis 1.3.7 (vendor string `Xiph.Org libVorbis I 20200704` in the Ogg headers)
- Canonical concept: none; sound has no concept stage (`concept/` is not used)
- Inputs: none. The script reads no file
- Similarity check: not applicable to these short synthetic effects beyond S-3 of `docs/sound.md`; not done
- Human work: none. RC-5 stays open as for scripted image work; the script is in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. Notes from the agent for whoever signs: no voice, no words, no casino or slot cue (no coin, reel or jackpot sound), no sudden loud sound (momentary loudness capped at -14 LUFS, section Loudness)
- Cultural review: not done (docs/localization.md 6.1 step 3, S-4)

## Synthesis

Run on 2026-11-15 by Claude Code, from the repository root:

```bash
python -I assets-src/sound/sfx/work/sfx.py assets-src/sound/sfx/work assets-src/sound/sfx/export
```

Two more runs into empty folders wrote byte-identical files (sha256 of all 34 files compared), and so did the run into this folder.

| Model | Used by | How |
|---|---|---|
| Tile clack | `tile_place_*`, `tile_draw` | Five damped sine modes near 1.85, 2.93, 4.41, 6.12 and 8.35 kHz (jittered by ±6% per variant, decays of about 7 to 52 ms) for the ivory body, a 2 to 12 kHz noise burst of 1.2 ms decay for the contact, and for a placement a low carpet thud (165 and 240 Hz modes, low-passed noise). A second, smaller clack 22 to 34 ms later is the bounce |
| Knock | `pass` | Two knuckle knocks on the table 160 ms apart: modes at 210, 428, 780 and 1350 Hz with a low-passed noise burst |
| Wood tick | `turn_warning` | Modes at 1580, 3191 and 4800 Hz with a short bright noise burst |
| Chime | `score`, `combo_*`, `hand_end`, `match_win` | Additive partials at ratios 1, 2, 3, 4.07, 5.2 and 6.8, higher partials decaying faster, 2 ms attack, a faint hammer noise |
| Pluck | `hand_end`, `match_win`, `match_lose` | Additive plucked string: harmonics to 12 kHz with slight inharmonicity, weights of a pluck at 13% (20% for `match_lose`) of the string, decay shorter for higher partials |
| Pad | `match_win`, `match_lose` | Two detuned voices per note, eight harmonics each, slow attack and release, low in the mix |

Notes are on D major pentatonic (D, E, F#, A, B) so the chimes sit in the key of both music loops. `score` is A5 then E6; `combo_1` to `combo_5` are D6, E6, F#6, A6 and B6 over the octave below, brighter with each step, with the fifth above from step 3. `hand_end` and `match_win` are rising D major arpeggios ending in chimes; `match_lose` is a soft falling F#4, D4, B3 under a 2.5 kHz low-pass.

Every sound starts at sample value 0, ends with a raised-cosine fade to 0, and is high-passed at 20 Hz before levelling.

## Loudness

Peak target per sound, then the gain is lowered if the highest momentary loudness (400 ms, BS.1770-4 K-weighting) would exceed -14 LUFS (`docs/sound.md` section 4). Measured on 2026-11-15 on the committed files; true peak by 4 times oversampling; the Ogg column is the decoded Ogg file.

| Name | Group | Seed | Peak target dBTP | Length s | True peak dBTP | Momentary max LUFS | Ogg true peak dBTP | WAV bytes | Ogg bytes |
|---|---|---|---|---|---|---|---|---|---|
| tile_place_01 | sfx | 1771 | -1 | 0.30 | -1.0 | -22.0 | -1.2 | 28,844 | 6,138 |
| tile_place_02 | sfx | 1772 | -1 | 0.30 | -1.0 | -21.1 | -0.9 | 28,844 | 6,188 |
| tile_place_03 | sfx | 1773 | -1 | 0.30 | -1.0 | -20.8 | -1.3 | 28,844 | 6,134 |
| tile_place_04 | sfx | 1774 | -1 | 0.30 | -1.0 | -20.6 | -0.9 | 28,844 | 6,130 |
| tile_draw | sfx | 1780 | -3 | 0.36 | -3.0 | -19.9 | -3.2 | 34,604 | 6,249 |
| pass | sfx | 1790 | -2 | 0.45 | -2.0 | -19.4 | -2.4 | 43,244 | 5,905 |
| turn_warning | sfx | 1800 | -4 | 0.12 | -4.0 | -22.3 | -4.3 | 11,564 | 4,332 |
| score | sfx | 1810 | -1 | 0.90 | -3.4 | -14.0 | -3.5 | 86,444 | 8,600 |
| combo_1 | sfx | 1821 | -1 | 0.75 | -4.3 | -14.0 | -4.4 | 72,044 | 7,448 |
| combo_2 | sfx | 1822 | -1 | 0.75 | -4.3 | -14.0 | -3.9 | 72,044 | 7,460 |
| combo_3 | sfx | 1823 | -1 | 0.75 | -4.7 | -14.0 | -4.9 | 72,044 | 8,244 |
| combo_4 | sfx | 1824 | -1 | 0.75 | -4.7 | -14.0 | -4.5 | 72,044 | 8,648 |
| combo_5 | sfx | 1825 | -1 | 0.75 | -4.3 | -14.0 | -4.6 | 72,044 | 8,370 |
| hand_end | sfx | 1830 | -1 | 1.60 | -2.4 | -14.0 | -2.3 | 153,644 | 18,393 |
| match_win | sfx | 1840 | -1 | 2.80 | -3.7 | -14.0 | -3.8 | 268,844 | 28,076 |
| match_lose | sfx | 1850 | -1 | 2.20 | -3.3 | -14.0 | -3.4 | 211,244 | 19,906 |
| ui_click | ui | 1860 | -6 | 0.06 | -6.0 | -29.2 | -6.9 | 5,804 | 4,224 |

Lower targets keep the frequent or attention sounds (`ui_click`, `turn_warning`, `tile_draw`, `pass`) below the tile clack. The seed is the NumPy `default_rng` seed of the sound's function in `sfx.py`.

## Checks

Done on 2026-11-15 by Claude Code; no listening was possible, so the checks are numeric and visual.

| Check | Result |
|---|---|
| Clipping | No 16-bit sample at ±32767 in any WAV; no decoded Ogg sample at or over full scale |
| DC offset | 20 Hz high-pass before levelling; the whole-file mean is under 0.001 of full scale for every sound, and every file starts and ends at sample value 0 |
| Clicks | Waveform and spectrogram PNGs of every file inspected: no broadband line other than the intended attacks. A first render had clicks where the knock and clack layers were cut off at the end of their buffers (`pass`, `tile_draw`); both layers now fade out and the lines are gone |
| Determinism | Three runs, byte-identical |
| Loudness meter | The meter code measured a stereo 1 kHz sine at -23 dBFS as -22.99 LUFS |

Not done: listening on a device (S-2), the Unity import and its sizes (`docs/sound.md` section 6), self-check and cultural review.

## Files

| File | sha256 | What it is |
|---|---|---|
| work/sfx.py | `0bfa2b068404fadb7f0e641a8ed20aa951897089a0e01efddeb91938d6a8e143` | The script, written by Claude Code on 2026-11-15 |
| work/sfx_<name>.wav | Git LFS oid of each file | Master: WAV 48 kHz 16-bit PCM mono, the source for the Unity import |
| export/<name>.ogg | Git LFS oid of each file | Ogg Vorbis, mono 48 kHz, libsndfile compression level 0.6, stream serial 177 |

# sound/music

The two music loops of `docs/sound.md` (#177), `lobby` and `match`, composed and rendered by one script, `work/music.py`. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: handmade. No file came from a generator; every sample is computed by `work/music.py` (`docs/asset-workflow.md` section 3, `docs/sound.md` section 7)
- Authors: composition rules, arrangement, synthesis models and mix by an AI agent (Claude Code) on 2026-11-15. No person has composed, played or listened to this music
- Generator: none. No music or audio generator, no sample, loop, MIDI file, preset or sound font
- Terms: not applicable, no generator or third-party material was used. The tools are Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3, soundfile 0.13.1 with libsndfile 1.2.2 and libvorbis 1.3.7 (vendor string `Xiph.Org libVorbis I 20200704` in the Ogg headers)
- Canonical concept: none; sound has no concept stage
- Inputs: none. The script reads no file, and no existing melody was transcribed or quoted
- Similarity check: not done. The melodies are seeded random walks (below), so a resemblance to an existing work would be chance; a person listens for it (S-3 of `docs/sound.md`)
- Human work: none. RC-5 stays open as for scripted image work; the script is in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. Notes from the agent for whoever signs: instrumental only, no voice or words, no national, religious or casino cue in the instrumentation, calm tempo and level
- Cultural review: not done (docs/localization.md 6.1 step 3, S-4)

## Composition and rendering

Run on 2026-11-15 by Claude Code, from the repository root:

```bash
python -I assets-src/sound/music/work/music.py assets-src/sound/music/work assets-src/sound/music/export
```

Two more runs into empty folders wrote byte-identical files (sha256 of all 4 files compared), and so did the run into this folder.

| | lobby | match |
|---|---|---|
| Tempo, metre | 80 BPM, 4/4 (a beat is 36,000 samples) | 96 BPM, 4/4 (30,000 samples) |
| Length | 24 bars, 72.000 s, 3,456,000 samples | 24 bars, 60.000 s, 2,880,000 samples |
| Key, chords (one per bar, eight bars played three times) | D major: Dmaj7, Bm7, Gmaj7, A7sus4, F#m7, Em7, Gmaj7, A6 | B minor: Bm7, Gmaj7, Em7, F#m7, Bm7, Gmaj7, Em7, A6 |
| Pad | Every bar, the chord voicing, slow attack | Same |
| Bass | Plucked, root on beat 1 and fifth on beat 3, partials to 3 kHz | Same |
| Keys | Two-operator FM tone, the chord at the start of every bar, notes 2.5 ms apart | Same |
| Arpeggio | Plucked eighth notes an octave above the voicing, partials to 7 kHz, bars 1 to 8 and 17 to 24 | Every bar |
| Melody | Plucked, partials to 9 kHz, bars 9 to 24, range A4 to D6 | FM keys, bars 9 to 20, range D4 to F#5, below the chime range of the sfx |
| Percussion | None | A soft band-passed noise shaker on eighth notes, accents on the beat |
| Seed | 2026111501 | 2026111502 |

- Melody: notes of D major pentatonic within the range, moved by a seeded random walk of -2 to +2 scale steps; a note on beat 1 or 3 is moved to the nearest chord tone; the rhythm of each bar is drawn from six patterns, and every second bar from three longer cadence patterns, so phrases are two bars long. All draws come from NumPy `default_rng(seed)`.
- Reverb: a synthetic stereo response (seeded noise, seed + 1000, 15 ms pre-delay, decay to -60 dB in 1.8 s, 2.4 s long, 6 kHz low-pass), applied by circular convolution, mixed at 0.22.
- Loop: every note, release and reverb tail that passes the end of the loop is added back at its start, so the file is one period of a periodic signal.
- Level: mean removed, gain to -16.0 LUFS integrated, then a circular look-ahead limiter at -1.5 dBFS that would lower the gain only where needed. It did not act on either loop: both peak below its ceiling at the target loudness.

## Loudness and seam

Measured on 2026-11-15 on the committed files with BS.1770-4 gating (`docs/sound.md` section 4); the Ogg rows are the decoded Ogg files.

| File | Channels | Integrated LUFS | Momentary max LUFS | True peak dBTP | Seam step | Median step | Bytes |
|---|---|---|---|---|---|---|---|
| work/music_lobby.wav | 2 | -16.0 | -11.9 | -2.8 | 0.0101 | 0.0084 | 13,824,044 |
| work/music_match.wav | 2 | -16.0 | -12.2 | -3.7 | 0.0034 | 0.0084 | 11,520,044 |
| export/lobby.ogg | 2 | -16.04 | -11.9 | -2.8 | 0.0132 | 0.0084 | 829,766 (92 kbit/s) |
| export/match.ogg | 2 | -16.04 | -12.3 | -3.9 | 0.0057 | 0.0083 | 859,672 (115 kbit/s) |

The seam step is the largest change between the last and the first sample across both channels, on a full scale of 1; the median step is that of all neighbouring samples of the file. Every seam step lies inside the file's ordinary steps (between the 13th and the 72nd percentile), and a plot of 48 samples on each side of the seam shows one continuous curve for all four files.

| Check | Result |
|---|---|
| Clipping | No 16-bit sample at ±32767; no decoded Ogg sample at or over full scale |
| DC offset | Mean removed; the remaining mean is under 2e-8 of full scale in the WAV files. The decoded Ogg files have exactly the sample count of the masters, so no codec padding enters the loop |
| Clicks | Waveform and spectrogram PNGs inspected: no broadband line at the seam or elsewhere besides note attacks and the shaker |
| Determinism | Three runs, byte-identical |

Not done: listening on a device (S-2), the seam after Unity's own encoding, the Unity import and its sizes, the similarity listening check (S-3), self-check and cultural review.

## Files

| File | sha256 | What it is |
|---|---|---|
| work/music.py | `ff0cbf9d3abb3a7dc6a2e384ac2254fe44e30015210aebdbd0cb1e676dd603c8` | The script, written by Claude Code on 2026-11-15 |
| work/music_lobby.wav, work/music_match.wav | Git LFS oid of each file | Masters: WAV 48 kHz 16-bit PCM stereo, the source for the Unity import |
| export/lobby.ogg, export/match.ogg | Git LFS oid of each file | Ogg Vorbis, stereo 48 kHz, libsndfile compression level 0.6, stream serial 177 |

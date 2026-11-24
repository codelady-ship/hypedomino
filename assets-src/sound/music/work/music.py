import io
import sys
from pathlib import Path

import numpy as np
import soundfile as sf
from scipy import ndimage, signal

RATE = 48000
LUFS_TARGET = -16.0
TRUE_PEAK_CEILING_DBTP = -1.0
OGG_COMPRESSION_LEVEL = 0.6
OGG_SERIAL = 177
OGG_BLOCK = 4096
K_SHELF = ([1.53512485958697, -2.69169618940638, 1.19839281085285], [1.0, -1.69065929318241, 0.73248077421585])
K_HIGHPASS = ([1.0, -2.0, 1.0], [1.0, -1.99004745483398, 0.99007225036621])
PENTATONIC = {2, 4, 6, 9, 11}
RHYTHMS = [
    [(0, 3), (3, 1), (4, 4)],
    [(0, 2), (2, 2), (4, 2), (6, 2)],
    [(0, 6), (6, 2)],
    [(2, 2), (4, 4)],
    [(0, 4), (4, 3)],
    [(1, 1), (2, 2), (4, 4)],
]
CADENCE_RHYTHMS = [[(0, 8)], [(0, 2), (2, 6)], [(0, 4), (4, 4)]]
CHORDS = {
    "dmaj7": (38, [57, 61, 62, 66]),
    "bm7": (47, [57, 59, 62, 66]),
    "gmaj7": (43, [55, 59, 62, 66]),
    "a7sus4": (45, [55, 57, 62, 64]),
    "fsm7": (42, [57, 61, 64, 66]),
    "em7": (40, [55, 59, 62, 64]),
    "a6": (45, [57, 61, 64, 66]),
}
SONGS = {
    "lobby": {
        "bpm": 80,
        "progression": ["dmaj7", "bm7", "gmaj7", "a7sus4", "fsm7", "em7", "gmaj7", "a6"] * 3,
        "melody_bars": range(8, 24),
        "melody_range": (69, 86),
        "melody_voice": "pluck",
        "arpeggio_bars": [bar for bar in range(24) if bar < 8 or bar >= 16],
        "shaker": False,
        "seed": 2026111501,
    },
    "match": {
        "bpm": 96,
        "progression": ["bm7", "gmaj7", "em7", "fsm7", "bm7", "gmaj7", "em7", "a6"] * 3,
        "melody_bars": range(8, 20),
        "melody_range": (62, 78),
        "melody_voice": "keys",
        "arpeggio_bars": list(range(24)),
        "shaker": True,
        "seed": 2026111502,
    },
}


def hz(midi):
    return 440.0 * 2.0 ** ((midi - 69) / 12.0)


def times(samples):
    return np.arange(samples) / RATE


def fades(sound, attack, release):
    attack_count = int(attack * RATE)
    release_count = int(release * RATE)
    sound[:attack_count] *= 0.5 - 0.5 * np.cos(np.linspace(0.0, np.pi, attack_count))
    sound[-release_count:] *= 0.5 + 0.5 * np.cos(np.linspace(0.0, np.pi, release_count))
    return sound


def pluck(freq, seconds, rng, position=0.13, brightness=12000.0):
    t = times(int(seconds * RATE))
    total = np.zeros_like(t)
    harmonic = 1
    while harmonic * freq < brightness:
        partial = harmonic * freq * np.sqrt(1.0 + 0.0002 * harmonic * harmonic)
        weight = abs(np.sin(np.pi * harmonic * position)) / harmonic
        decay = 1.4 / (1.0 + (harmonic * freq / 1800.0) ** 1.3)
        total += weight * np.exp(-t / decay) * np.sin(2.0 * np.pi * partial * t + rng.uniform(0.0, 0.2))
        harmonic += 1
    return fades(total, 0.003, 0.05)


def keys(freq, seconds):
    t = times(int(seconds * RATE))
    index = 1.6 * np.exp(-t / 0.35) + 0.25
    modulator = np.sin(2.0 * np.pi * freq * t)
    tone = np.sin(2.0 * np.pi * freq * t + index * modulator) * np.exp(-t / 1.6)
    return fades(tone, 0.004, 0.08)


def pad(freqs, seconds, attack, release):
    t = times(int(seconds * RATE))
    total = np.zeros_like(t)
    for freq in freqs:
        for detune in (-0.0023, 0.0023):
            for harmonic in range(1, 9):
                total += np.exp(-harmonic / 3.0) / harmonic * np.sin(2.0 * np.pi * freq * (1.0 + detune) * harmonic * t)
    envelope = np.minimum(1.0, t / attack) * np.minimum(1.0, (seconds - t) / release)
    return total * np.clip(envelope, 0.0, 1.0)


def shaker(rng, accent):
    t = times(int(0.08 * RATE))
    raw = rng.standard_normal(len(t)) * np.exp(-t / 0.018)
    band = signal.sosfilt(signal.butter(2, [5000.0, 12000.0], btype="bandpass", fs=RATE, output="sos"), raw)
    return fades(accent * band, 0.002, 0.02)


def add(mix, start, sound, pan):
    angle = (pan + 1.0) * np.pi / 4.0
    stereo = np.stack([np.cos(angle) * sound, np.sin(angle) * sound], axis=1)
    length = len(mix)
    start %= length
    first = min(len(stereo), length - start)
    mix[start : start + first] += stereo[:first]
    rest = stereo[first:]
    while len(rest):
        chunk = rest[:length]
        mix[: len(chunk)] += chunk
        rest = rest[length:]


def melody(rng, chord_names, low, high):
    pitches = [midi for midi in range(low, high + 1) if midi % 12 in PENTATONIC]
    position = len(pitches) // 2
    notes = []
    for bar, name in enumerate(chord_names):
        cadence = bar % 2 == 1
        bank = CADENCE_RHYTHMS if cadence else RHYTHMS
        pattern = bank[rng.integers(len(bank))]
        chord_classes = {midi % 12 for midi in CHORDS[name][1]}
        for start, length in pattern:
            position = int(np.clip(position + rng.choice([-2, -1, -1, 0, 1, 1, 2]), 0, len(pitches) - 1))
            strong = start in (0, 4)
            if strong and pitches[position] % 12 not in chord_classes:
                candidates = [index for index, midi in enumerate(pitches) if midi % 12 in chord_classes]
                if candidates:
                    position = min(candidates, key=lambda index: abs(index - position))
            notes.append((bar, start, length, pitches[position]))
    return notes


def arpeggio_order(bar):
    return [0, 1, 2, 3, 2, 1, 3, 2] if bar % 2 == 0 else [0, 2, 1, 3, 1, 2, 0, 3]


def compose(song):
    rng = np.random.default_rng(song["seed"])
    beat = int(round(60.0 / song["bpm"] * RATE))
    eighth = beat // 2
    bar_samples = 4 * beat
    progression = song["progression"]
    mix = np.zeros((bar_samples * len(progression), 2))
    bar_seconds = bar_samples / RATE
    for bar, name in enumerate(progression):
        bass, voicing = CHORDS[name]
        start = bar * bar_samples
        add(mix, start, 0.05 * pad([hz(midi) for midi in voicing], bar_seconds + 1.0, 0.8, 1.0), 0.0)
        add(mix, start, 0.5 * pluck(hz(bass), 2.2, rng, 0.25, 3000.0), -0.1)
        add(mix, start + 2 * beat, 0.35 * pluck(hz(bass + 7), 1.8, rng, 0.25, 3000.0), -0.1)
        for note_index, midi in enumerate(voicing):
            add(mix, start + note_index * 120, 0.09 * keys(hz(midi), 2.4), -0.35 + 0.2 * note_index)
        if bar in song["arpeggio_bars"]:
            for step, voice in enumerate(arpeggio_order(bar)):
                velocity = 0.16 if step % 2 == 0 else 0.11
                add(mix, start + step * eighth, velocity * pluck(hz(voicing[voice] + 12), 1.4, rng, 0.13, 7000.0), 0.4)
        if song["shaker"]:
            for step in range(8):
                add(mix, start + step * eighth, 0.05 * shaker(rng, 1.0 if step % 2 == 0 else 0.55), 0.25)
    melody_bars = list(song["melody_bars"])
    low, high = song["melody_range"]
    for bar, start, length, midi in melody(rng, [progression[bar] for bar in melody_bars], low, high):
        at = (melody_bars[0] + bar) * bar_samples + start * eighth
        seconds = length * eighth / RATE + 0.6
        if song["melody_voice"] == "pluck":
            add(mix, at, 0.3 * pluck(hz(midi), seconds + 0.8, rng, 0.18, 9000.0), -0.15)
        else:
            add(mix, at, 0.22 * keys(hz(midi), seconds + 0.8), -0.15)
    return mix


def reverb(mix, seed):
    rng = np.random.default_rng(seed)
    t = times(int(2.4 * RATE))
    lowpass = signal.butter(2, 6000.0, btype="lowpass", fs=RATE, output="sos")
    wet = np.zeros_like(mix)
    for channel in range(2):
        tail = signal.sosfilt(lowpass, rng.standard_normal(len(t)) * np.exp(-6.9 * t / 1.8))
        response = np.concatenate([np.zeros(int(0.015 * RATE)), tail])
        response /= np.sqrt(np.sum(response ** 2))
        spectrum = np.fft.rfft(mix[:, channel]) * np.fft.rfft(response, len(mix))
        wet[:, channel] = np.fft.irfft(spectrum, len(mix))
    return mix + 0.22 * wet


def integrated_lufs(mix):
    looped = np.concatenate([mix, mix])
    weighted = signal.lfilter(*K_HIGHPASS, signal.lfilter(*K_SHELF, looped, axis=0), axis=0)[len(mix) :]
    block = int(0.4 * RATE)
    step = int(0.1 * RATE)
    powers = np.array([np.sum(np.mean(weighted[start : start + block] ** 2, axis=0)) for start in range(0, len(weighted) - block + 1, step)])
    loudness = -0.691 + 10.0 * np.log10(powers + 1e-20)
    gated = powers[loudness > -70.0]
    relative = -0.691 + 10.0 * np.log10(np.mean(gated)) - 10.0
    gated = powers[(loudness > -70.0) & (loudness > relative)]
    return -0.691 + 10.0 * np.log10(np.mean(gated))


def true_peak_dbtp(mix):
    wrapped = np.concatenate([mix[-64:], mix, mix[:64]])
    return 20.0 * np.log10(np.max(np.abs(signal.resample_poly(wrapped, 4, 1, axis=0))))


def limited(mix, ceiling_db):
    ceiling = 10.0 ** (ceiling_db / 20.0)
    level = np.max(np.abs(mix), axis=1)
    needed = np.minimum(1.0, ceiling / np.maximum(level, 1e-9))
    width = int(0.004 * RATE) | 1
    gain = ndimage.uniform_filter1d(ndimage.minimum_filter1d(needed, width, mode="wrap"), width, mode="wrap")
    return mix * gain[:, None]


def mastered(mix):
    mix = mix - np.mean(mix, axis=0)
    gain = 10.0 ** ((LUFS_TARGET - integrated_lufs(mix)) / 20.0)
    ceiling_db = TRUE_PEAK_CEILING_DBTP - 0.5
    for _ in range(12):
        result = limited(mix * gain, ceiling_db)
        loudness = integrated_lufs(result)
        if true_peak_dbtp(result) > TRUE_PEAK_CEILING_DBTP:
            ceiling_db -= 0.2
            continue
        if abs(loudness - LUFS_TARGET) < 0.05:
            return result
        gain *= 10.0 ** ((LUFS_TARGET - loudness) / 20.0)
    raise RuntimeError("loudness target not reached")


def ogg_crc_table():
    table = []
    for index in range(256):
        value = index << 24
        for _ in range(8):
            value = ((value << 1) ^ 0x04C11DB7) if value & 0x80000000 else value << 1
            value &= 0xFFFFFFFF
        table.append(value)
    return table


def with_fixed_serial(ogg):
    table = ogg_crc_table()
    data = bytearray(ogg)
    position = 0
    while position < len(data):
        if data[position : position + 4] != b"OggS":
            raise ValueError("not an Ogg page")
        segments = data[position + 26]
        end = position + 27 + segments + sum(data[position + 27 : position + 27 + segments])
        data[position + 14 : position + 18] = OGG_SERIAL.to_bytes(4, "little")
        data[position + 22 : position + 26] = bytes(4)
        crc = 0
        for byte in data[position:end]:
            crc = ((crc << 8) & 0xFFFFFFFF) ^ table[(crc >> 24) ^ byte]
        data[position + 22 : position + 26] = crc.to_bytes(4, "little")
        position = end
    return bytes(data)


def write(name, mix, work, export):
    pcm = np.round(np.clip(mix, -1.0, 1.0) * 32767.0).astype(np.int16)
    sf.write(work / f"music_{name}.wav", pcm, RATE, subtype="PCM_16")
    buffer = io.BytesIO()
    samples = pcm.astype(np.float32) / 32768.0
    with sf.SoundFile(buffer, "w", RATE, 2, format="OGG", subtype="VORBIS", compression_level=OGG_COMPRESSION_LEVEL) as encoder:
        for start in range(0, len(samples), OGG_BLOCK):
            encoder.write(samples[start : start + OGG_BLOCK])
    (export / f"{name}.ogg").write_bytes(with_fixed_serial(buffer.getvalue()))


def main(work, export):
    work.mkdir(parents=True, exist_ok=True)
    export.mkdir(parents=True, exist_ok=True)
    for name, song in SONGS.items():
        write(name, mastered(reverb(compose(song), song["seed"] + 1000)), work, export)


if __name__ == "__main__":
    main(Path(sys.argv[1]), Path(sys.argv[2]))

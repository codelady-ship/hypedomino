import io
import sys
from pathlib import Path

import numpy as np
import soundfile as sf
from scipy import signal

RATE = 48000
OGG_COMPRESSION_LEVEL = 0.6
OGG_SERIAL = 177
OGG_BLOCK = 4096
MOMENTARY_CEILING_LUFS = -14.0
K_SHELF = ([1.53512485958697, -2.69169618940638, 1.19839281085285], [1.0, -1.69065929318241, 0.73248077421585])
K_HIGHPASS = ([1.0, -2.0, 1.0], [1.0, -1.99004745483398, 0.99007225036621])
PENTATONIC_D6 = [1174.66, 1318.51, 1479.98, 1760.0, 1975.53]


def times(seconds):
    return np.arange(int(round(seconds * RATE))) / RATE


def place(buffer, sound, at_seconds, gain=1.0):
    start = int(round(at_seconds * RATE))
    end = min(len(buffer), start + len(sound))
    buffer[start:end] += gain * sound[: end - start]
    return buffer


def fade_out(sound, seconds):
    count = int(round(seconds * RATE))
    ramp = 0.5 + 0.5 * np.cos(np.linspace(0.0, np.pi, count))
    sound[-count:] *= ramp
    return sound


def fade_in(sound, seconds):
    count = int(round(seconds * RATE))
    sound[:count] *= 0.5 - 0.5 * np.cos(np.linspace(0.0, np.pi, count))
    return sound


def modes(freqs, decays, amps, seconds):
    t = times(seconds)
    total = np.zeros_like(t)
    for freq, decay, amp in zip(freqs, decays, amps):
        total += amp * np.exp(-t / decay) * np.sin(2.0 * np.pi * freq * t)
    return total


def noise_burst(rng, seconds, decay, low, high):
    t = times(seconds)
    raw = rng.standard_normal(len(t)) * np.exp(-t / decay)
    if high >= RATE / 2:
        sos = signal.butter(2, low, btype="highpass", fs=RATE, output="sos")
    elif low <= 0:
        sos = signal.butter(2, high, btype="lowpass", fs=RATE, output="sos")
    else:
        sos = signal.butter(2, [low, high], btype="bandpass", fs=RATE, output="sos")
    return fade_in(signal.sosfilt(sos, raw), 0.0005)


def clack(rng, seconds, scale, with_thud):
    jitter = rng.uniform(0.94, 1.06)
    freqs = np.array([1850.0, 2930.0, 4410.0, 6120.0, 8350.0]) * jitter
    decays = np.array([0.045, 0.030, 0.020, 0.012, 0.008]) * rng.uniform(0.85, 1.15)
    amps = np.array([1.0, 0.7, 0.5, 0.35, 0.2]) * rng.uniform(0.8, 1.2, 5)
    body = modes(freqs, decays, amps, seconds)
    body += 0.8 * noise_burst(rng, seconds, 0.0012, 2000.0, 12000.0)
    if with_thud:
        low = rng.uniform(0.92, 1.08)
        body += modes([165.0 * low, 240.0 * low], [0.035, 0.025], [0.6, 0.35], seconds)
        body += 4.0 * noise_burst(rng, seconds, 0.02, 0.0, 600.0)
    return fade_out(scale * body, 0.02)


def tile_place(variant):
    rng = np.random.default_rng(1770 + variant)
    length = 0.3
    sound = clack(rng, length, 1.0, True)
    bounce = clack(rng, length, rng.uniform(0.18, 0.3), False)
    place(sound, bounce, rng.uniform(0.022, 0.034))
    return fade_out(sound, 0.03)


def tile_draw():
    rng = np.random.default_rng(1780)
    length = 0.36
    slide_t = times(0.14)
    slide = rng.standard_normal(len(slide_t)) * np.sin(np.pi * slide_t / 0.14) ** 2
    slide = signal.sosfilt(signal.butter(2, [800.0, 5000.0], btype="bandpass", fs=RATE, output="sos"), slide)
    sound = np.zeros(len(times(length)))
    place(sound, 0.25 * slide, 0.0)
    place(sound, clack(rng, 0.22, 0.5, False), 0.13)
    return fade_out(sound, 0.03)


def knock(rng, seconds):
    body = modes([210.0, 428.0, 780.0, 1350.0], [0.06, 0.04, 0.025, 0.015], [1.0, 0.6, 0.35, 0.2], seconds)
    return fade_out(body + 1.5 * noise_burst(rng, seconds, 0.003, 0.0, 2000.0), 0.03)


def pass_knock():
    rng = np.random.default_rng(1790)
    length = 0.45
    sound = np.zeros(len(times(length)))
    place(sound, knock(rng, 0.28), 0.0)
    place(sound, knock(rng, 0.28), 0.16, 0.85)
    return fade_out(sound, 0.04)


def turn_warning():
    rng = np.random.default_rng(1800)
    sound = modes([1580.0, 3191.0, 4800.0], [0.03, 0.015, 0.008], [1.0, 0.4, 0.2], 0.12)
    sound += 0.3 * noise_burst(rng, 0.12, 0.001, 3000.0, 12000.0)
    return fade_out(sound, 0.02)


def chime(freq, seconds, brightness, rng, ring=0.25):
    t = times(seconds)
    ratios = [1.0, 2.0, 3.0, 4.07, 5.2, 6.8]
    weights = [1.0, 0.45, 0.25, 0.15, 0.08, 0.05]
    total = np.zeros_like(t)
    for index, (ratio, weight) in enumerate(zip(ratios, weights)):
        partial = freq * ratio
        if partial >= 0.45 * RATE:
            continue
        decay = ring / (1.0 + 0.6 * index)
        total += weight * brightness ** index * np.exp(-t / decay) * np.sin(2.0 * np.pi * partial * t)
    total += 0.05 * noise_burst(rng, seconds, 0.0015, 5000.0, RATE)
    return fade_out(fade_in(total, 0.002), 0.05)


def pluck(freq, seconds, rng, position=0.13):
    t = times(seconds)
    total = np.zeros_like(t)
    harmonic = 1
    while harmonic * freq < 12000.0:
        partial = harmonic * freq * np.sqrt(1.0 + 0.0002 * harmonic * harmonic)
        weight = abs(np.sin(np.pi * harmonic * position)) / harmonic
        decay = 1.4 / (1.0 + (harmonic * freq / 1800.0) ** 1.3)
        total += weight * np.exp(-t / decay) * np.sin(2.0 * np.pi * partial * t + rng.uniform(0.0, 0.2))
        harmonic += 1
    return fade_out(fade_in(total, 0.003), 0.05)


def pad(freqs, seconds, attack, release):
    t = times(seconds)
    total = np.zeros_like(t)
    for freq in freqs:
        for detune in (-0.0023, 0.0023):
            for harmonic in range(1, 9):
                total += np.exp(-harmonic / 3.0) / harmonic * np.sin(2.0 * np.pi * freq * (1.0 + detune) * harmonic * t)
    envelope = np.minimum(1.0, t / attack) * np.minimum(1.0, (seconds - t) / release)
    return total * np.clip(envelope, 0.0, 1.0)


def score():
    rng = np.random.default_rng(1810)
    sound = np.zeros(len(times(0.9)))
    place(sound, chime(880.0, 0.83, 0.8, rng), 0.0)
    place(sound, chime(1318.51, 0.83, 0.8, rng), 0.07)
    return fade_out(sound, 0.05)


def combo(step):
    rng = np.random.default_rng(1820 + step)
    note = PENTATONIC_D6[step - 1]
    brightness = 0.6 + 0.1 * step
    sound = np.zeros(len(times(0.75)))
    place(sound, chime(note / 2.0, 0.7, brightness, rng), 0.0, 0.3)
    place(sound, chime(note, 0.7, brightness, rng), 0.0)
    if step >= 3:
        place(sound, chime(note * 1.5, 0.6, brightness, rng), 0.05, 0.25)
    return fade_out(sound, 0.05)


def hand_end():
    rng = np.random.default_rng(1830)
    sound = np.zeros(len(times(1.6)))
    for index, freq in enumerate([293.66, 369.99, 440.0, 587.33]):
        place(sound, pluck(freq, 1.3, rng), 0.09 * index, 0.8)
    place(sound, chime(1174.66, 1.2, 0.6, rng, 0.6), 0.27, 0.35)
    return fade_out(sound, 0.15)


def match_win():
    rng = np.random.default_rng(1840)
    sound = np.zeros(len(times(2.8)))
    for index, freq in enumerate([293.66, 440.0, 587.33, 739.99, 880.0]):
        place(sound, pluck(freq, 1.8, rng), 0.11 * index, 0.8)
    place(sound, pluck(146.83, 2.0, rng), 0.6, 0.9)
    for freq in [1174.66, 1479.98, 1760.0]:
        place(sound, chime(freq, 1.9, 0.7, rng, 0.8), 0.6, 0.3)
    place(sound, pad([293.66, 369.99, 440.0], 2.6, 0.9, 1.2), 0.1, 0.05)
    return fade_out(sound, 0.3)


def match_lose():
    rng = np.random.default_rng(1850)
    sound = np.zeros(len(times(2.2)))
    for index, freq in enumerate([369.99, 293.66, 246.94]):
        place(sound, pluck(freq, 1.6, rng, 0.2), 0.22 * index, 0.8)
    place(sound, pad([246.94, 293.66, 369.99], 2.0, 0.5, 1.0), 0.15, 0.04)
    lowpass = signal.butter(2, 2500.0, btype="lowpass", fs=RATE, output="sos")
    return fade_out(signal.sosfilt(lowpass, sound), 0.3)


def ui_click():
    rng = np.random.default_rng(1860)
    sound = modes([2600.0, 5200.0], [0.006, 0.003], [1.0, 0.4], 0.06)
    sound += 0.5 * noise_burst(rng, 0.06, 0.0015, 3000.0, 10000.0)
    return fade_out(sound, 0.015)


def true_peak_dbtp(sound):
    padded = np.concatenate([np.zeros(64), sound, np.zeros(64)])
    oversampled = signal.resample_poly(padded, 4, 1)
    return 20.0 * np.log10(np.max(np.abs(oversampled)) + 1e-12)


def max_momentary_lufs(sound):
    weighted = signal.lfilter(*K_HIGHPASS, signal.lfilter(*K_SHELF, np.concatenate([sound, np.zeros(RATE)])))
    window = int(0.4 * RATE)
    energy = np.concatenate([[0.0], np.cumsum(weighted ** 2)])
    power = (energy[window:] - energy[:-window]) / window
    return -0.691 + 10.0 * np.log10(np.max(power) + 1e-20)


def normalized(sound, peak_dbtp):
    sound = signal.sosfilt(signal.butter(2, 20.0, btype="highpass", fs=RATE, output="sos"), sound)
    sound = fade_out(fade_in(sound, 0.0005), 0.005)
    gain_db = min(peak_dbtp - true_peak_dbtp(sound), MOMENTARY_CEILING_LUFS - max_momentary_lufs(sound))
    return sound * 10.0 ** (gain_db / 20.0)


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


def write(name, sound, peak_dbtp, work, export):
    pcm = np.round(np.clip(normalized(sound, peak_dbtp), -1.0, 1.0) * 32767.0).astype(np.int16)
    sf.write(work / f"sfx_{name}.wav", pcm, RATE, subtype="PCM_16")
    buffer = io.BytesIO()
    samples = pcm.astype(np.float32) / 32768.0
    with sf.SoundFile(buffer, "w", RATE, 1, format="OGG", subtype="VORBIS", compression_level=OGG_COMPRESSION_LEVEL) as encoder:
        for start in range(0, len(samples), OGG_BLOCK):
            encoder.write(samples[start : start + OGG_BLOCK])
    (export / f"{name}.ogg").write_bytes(with_fixed_serial(buffer.getvalue()))


def sounds():
    yield from ((f"tile_place_{variant:02d}", tile_place(variant), -1.0) for variant in range(1, 5))
    yield "tile_draw", tile_draw(), -3.0
    yield "pass", pass_knock(), -2.0
    yield "turn_warning", turn_warning(), -4.0
    yield "score", score(), -1.0
    yield from ((f"combo_{step}", combo(step), -1.0) for step in range(1, 6))
    yield "hand_end", hand_end(), -1.0
    yield "match_win", match_win(), -1.0
    yield "match_lose", match_lose(), -1.0
    yield "ui_click", ui_click(), -6.0


def main(work, export):
    work.mkdir(parents=True, exist_ok=True)
    export.mkdir(parents=True, exist_ok=True)
    for name, sound, peak_dbtp in sounds():
        write(name, sound, peak_dbtp, work, export)


if __name__ == "__main__":
    main(Path(sys.argv[1]), Path(sys.argv[2]))

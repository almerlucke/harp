# Recordings: sampler and streamer

Two synths play recordings: `sampler` (SFZ instruments, and recordings cut into pieces on keys) and `streamer` (long recordings played from disk). Both are played by notes, like the other [synths](synths.md), and come out in stereo.

## Files

A file is a path. It is looked for **next to the patch** first, then in the folder harp runs in. wav, aiff, flac and ogg are read (whatever libsndfile reads). A missing file says where it looked:

```
there is no file rain.wav (looked in /music/patch/rain.wav and /home/me/rain.wav)
```

Files are read when the patch is built, and not again while their name, settings and the file itself do not change: a live edit that does not touch them does not read them again.

## sampler

```
bells = sampler(notes, "celesta.sfz", voices: 6)                       # an SFZ instrument, as it is
loop  = sampler(notes, "drums.wav", mode: "hits", args: [9, 0.08], first: 36)   # a recording cut at its hits
one   = sampler(notes, "bell.wav", first: 60)                          # the whole sound on key 60, at its own pitch
```

A `.sfz` file is played as the instrument it describes (keys, velocity layers, loops, round robin, release samples). Any other file is cut by `mode`, and its pieces are put on the keys from `first` up: a note of that pitch plays that piece. `harp check` says what a file was cut into:

```
66-bpm-uniq-percussions: 92 on keys 36-127
```

| `mode` | `args` | Pieces |
|---|---|---|
| `whole` (default) | `[highest key, key it is in tune at]` | the whole sound on a range of keys (default: only `first`) |
| `hits` | `[rise dB = 9, least gap s = 0.08, from s, to s]` | cut where a hit starts (the first piece is what comes before the first hit) |
| `chops` | `[rise, gap, from, to, longest piece s]` | the pieces at the hits, tuned to the key they are played on |
| `change` | `[sensitivity = 1.1, least gap s = 0.25]` | cut where the sound changes |
| `pauses` | `[depth dB = 30, least pause s = 0.3, longest piece s]` | cut at the pauses (a sung line) |
| `verses` | `[number of verses, depth dB = 20]` | verses and their refrains |
| `even` | `[number of pieces = 8]` | equal pieces |
| `notes` | `[from, to, pitch]` for each note | the sung notes you name, with their pitch, five keys apart |

Other arguments: `vol` (dB) and `pan` (-1 to 1) for the whole sound; `loop: 1.5` makes it go round while the key is down (the end fading into the start over that many seconds); `voices` (default 8); `mipmaps: 4` makes lower resolutions of each sound for cleaner high notes. They are filtered once and **kept on disk** (see below): the first load of a big instrument takes a second or more, the loads after that a fraction of one.

A note picks its pieces by key and by `velocity`. The voices of a sampler share their round robin turns and choke groups.

### The mipmap cache

The lower resolutions are kept in `~/.cache/harp/mipmaps` (`$XDG_CACHE_HOME`, or the folder in `HARP_CACHE_DIR`; `HARP_CACHE=off` keeps nothing). A file is kept by its path, size and time and the depth: a changed file is filtered again. They take about as much space as the sounds times the depth. `harp cache` says how big the folder is, `harp cache clear` empties it. The celesta of the Virtual Playing Orchestra with `mipmaps: 4`: 10 s to load without the cache (4 s on all cores), 0.23 s with it.

## streamer

```
rain = streamer(notes, "rain.wav", key: 60, loop: true)
```

The first half second of the file is kept in memory and the rest is read from disk while it plays, so a recording of any length costs the memory of half a second: an hour of rain, a piece of music. A note starts at once.

| Argument | Meaning |
|---|---|
| `key` | the note at which it plays at its own pitch (a note a fifth above plays it faster) |
| `loop` | go round between `loopStart` and `loopEnd` (seconds; the start and the end of the file by default) while the key is down |
| `hold` | a note off releases the sound over `release` seconds (otherwise it plays to its end, like a drum) |
| `speed`, `start`, `end` | play faster or slower; play a part of the file (seconds) |
| `vol`, `pan`, `voices` | dB; -1 to 1; default 4 |

The audio never waits for the disk. If the disk is too slow the sound has a gap (psalm counts them). Rendering to a file (`harp render`) waits for the reader a little per block, so a render of a patch with a streamer takes about as long as it plays.

## Live editing

The notes of a sampler or streamer (their rhythm, pitches, chords) can be edited without cutting what sounds. A change to the file, to `mode`, `args`, `first`, `vol`, `pan`, `loop`, `mipmaps` or any setting of the sound makes it a new instrument: what it was playing stops, and the file is read again only if its own settings changed. See [Live editing](live-editing.md).

## Example

`lang/examples/recordings.harp` plays a celesta (SFZ), percussion cut from a loop, and a long recording streamed under them, from the files in psalm's assets.

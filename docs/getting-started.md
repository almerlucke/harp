# Getting started

## Build

harp is one Go module (the `go.mod` in the root); it builds from the tagged versions of psalm, genny, sndfile and exbana that `go.mod` lists. To develop it together with local checkouts of those libraries, make a `go.work` that lists them (`go work init . ../psalm ../genny` and so on); it is not committed. From `lang/`:

```
go build -o harp ./cmd/harp
```

or run it without building: `go run ./cmd/harp ...`.

## Commands

| Command | What it does |
|---|---|
| `harp check patch.harp` | reads and checks a patch, prints its channels and tempo |
| `harp render patch.harp -o out.wav -seconds 10` | writes the sound to a WAV file (`-float` for 32 bit float, `-rate 44100`) |
| `harp play patch.harp` | plays it on the sound card, and again when you save the file ([live editing](live-editing.md)) |
| `harp compare patch.harp` | renders it and the SuperCollider code it translates, and says how alike they are ([SuperCollider](supercollider.md)) |
| `harp functions` | lists the functions (`-md` for markdown) |
| `harp presets [synth]` | lists the presets of the synths |
| `harp cache [clear]` | the folder of the kept mipmaps of recordings, and how big it is ([Recordings](recordings.md)) |

`play` takes `-from 4:24` (or `-from 264`: start that far into the piece), `-seconds N` (stop after N seconds), `-buffer 512`, `-rate 44100`, `-watch=false` (do not watch the file) and `-cpuprofile file`.

`-from` runs the generators on to that time without making a sound, so it starts at once, however long the piece is. What starts there is what the piece plays at that time, without what was sounding before it: a reverb starts empty, and a note that began before it (and still rings) is not heard.

## A first patch

Save as `first.harp`:

```harp
bpm = 110
seed = 1

melody = degrees(a3, minor, mutate([0 2 4 7 4 2], swap: 0.4))
voice  = notes(melody, euclid(16, 7), length: 0.6)

out = saw(voice.pitch)
  .lowPass(cutoff: lfo(0.2).range(400, 3000), res: 0.4)
  .gain(adsr(0.005, 0.15, 0.4, 0.2).on(voice.gate))
  .gain(0.3)
  .dup()
```

`harp play first.harp` plays it. Change the cutoff range or the rhythm and save: the sound changes without a restart.

What the patch says: a phrase in A minor that now and then swaps its notes (a generator); a rhythm of 7 hits in 16 steps (a generator); a saw that follows the notes, through a filter that an lfo sweeps, shaped by an envelope that the rhythm triggers; the result in stereo.

## Where next

[The language](language.md), then [Generators](generators.md).

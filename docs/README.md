# harp

A language for sound synthesis. Patches are text; generators (rhythms, phrases, notes) are part of the language; the engine is built for audio and a running patch can be edited while it plays.

harp combines three libraries: **psalm** (DSP blocks, effects, synths), **genny** (generators) and **exbana** (the parser).

## Where to look

| I want to... | Read |
|---|---|
| install it, play a first patch | [Getting started](getting-started.md) |
| know the syntax | [The language](language.md) |
| know what a value is, and what can be used where | [Types and channels](types.md) |
| make rhythms and phrases | [Generators](generators.md), [the library](generator-library.md) |
| make music from whole scores (MIDI files, fugues, canons) | [Scores](scores.md) |
| play notes on a synth of psalm | [Synths](synths.md), [Presets](presets.md) |
| play notes on an instrument I write myself | [play](play.md) |
| use the effects of psalm | [Effects](effects.md) |
| play recordings and SFZ instruments | [Recordings](recordings.md) |
| edit a patch while it plays | [Live editing](live-editing.md) |
| look up a function | [Reference](reference.md) |
| compare a patch with SuperCollider | [SuperCollider](supercollider.md) |
| see the example patches | [Examples](examples.md) |
| work on harp itself | [Architecture](architecture.md) |

The reference is made from the code (`harp functions -md`), so it does not go out of date. The other pages are written by hand and kept short: one subject each.

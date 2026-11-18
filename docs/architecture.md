# Architecture

For working on harp itself. Four packages, from the bottom up:

| Package | Is |
|---|---|
| `dsl` | the engine: a signal graph, kernels, the clock, generators as lanes, the optimizer, live swapping |
| `blocks` | psalm's filters, delays, reverb, oscillators and synths as nodes of that graph |
| `lang` | the text language: grammar (exbana), checker, evaluator, the functions |
| `lang/cmd/harp` | the command; `lang/sc` and `lang/compare` are the SuperCollider tools |

## Two rates

A signal is a graph of nodes, computed per sample. A generator (genny) makes a value per **step**. Generators never run in the audio loop: each registers a **lane** with a `pull` (control goroutine, ahead of the audio) and an `apply` (audio thread, on the sample the step starts). The control goroutine fills a ring with one record of values per step; the audio side applies the record of its step, which is a few assignments (`dsl/control.go`, `dsl/lift.go`, `dsl/events.go`).

## From text to sound

1. `lang/syntax.go` parses the text (exbana patterns) into a syntax tree.
2. `lang/eval.go` works it out from the inside out: the type of each part says what it is (`types.go`), functions come from `builtins*.go`. The result is a `dsl.Sig` for `out` and the clock.
3. `dsl/graph.go` compiles the graph into a flat list of kernels over preallocated 64-sample buffers (`Compile`); `dsl/optimize.go` first folds constants and affine chains and merges equal nodes (not used while watching).
4. `dsl/render.go` runs it: `Runner.Process` cuts a block where a step starts, so steps are sample-exact whatever the size of the device buffer. `dsl/play.go` connects it to an audio device.

Multichannel: every channel is its own node. `dsl/channels.go` has the rules (one channel goes with any number, otherwise equal counts).

## Custom nodes

`dsl.Custom` and `dsl.CustomN` make a node of your own (that is how `blocks` wraps psalm). A `Spec` has a `Key` that says what the node computes (equal key and inputs merge, and it is what live editing matches by) and `New`, which makes the state of one instance.

## Live editing

`dsl/swap.go` compiles the new graph *from* the one that plays: matching nodes (exact by structure, then by place), reusing their kernels and buffers; nodes that share state are kept or not together; a kept node whose input changed gets the new input copied into the buffer it reads. `dsl/control.go` hands the new program to the audio thread at its next step and translates the records that were made but not played. Patterns of notes (`dsl/events.go`) can have a field, the rhythm or the gate length replaced inside the running pattern. `lang/fingerprint.go` gives generators a fingerprint (the expression that made them), so the same generator keeps its place. The doc comment of `dsl/swap.go` is the place to start.

## Hosting psalm

Synths are voices of psalm played by one host (`blocks/synth.go`: notes from a lane, voices rendered in chunks of 64). Most take a note each voice and steal the oldest when they are busy; the organ is one instrument that every note goes to (the host's single mode, with a small adapter that makes it look like a voice), and the drum synth's voices are made together so that they can choke each other. A synth can have signals cabled into its voices' inputs (the wavetable's position, cutoff and resonance, and through its modulation matrix warp, detune, spread and sub): the host reads each at the start of every chunk and gives it to every voice. Rhythms are normally a boolean for each step; a pattern with rolls and triplets is a generator of hits that fall anywhere in a step (`dsl.Hit`), which `play` turns into events of 0 steps apart with an offset (`Event.Shift`), and a synth into notes that start late by a part of a step. The feel of `play` (swing, humanize, accent, grooves, lanes) is one rule of the step a note is on (`dsl.Groove`), so that a live edit of it can be given to the notes that were made already. Effects are modules of psalm played by another (`blocks/effects.go`): the host gives a module its input sockets, calls `Synthesize`, sends its settings as messages, and the settings, ranges and defaults come from the effect's own `Describe`, so `lang/effects.go` makes a function for each effect from that. `blocks/sampled.go` and `lang/sampled.go` play the sampler and streamer (the text of a sound is made of the arguments and read by psalm's own parser, and kept by name and file time; the mipmaps of the recordings are kept on disk by `sndfile`, `lang/cache.go` says where). The presets of the synths are found by `lang/internal/genpresets` (`presets_gen.go`).

## SuperCollider ports

`dsl/sc*.go` are ports of SuperCollider's unit generators, written from its source, so a translation can match it wave for wave (`scsaw`, filters, delays, `EnvGen`, `FreeVerb2`). They read their parameters once per 64-sample block, as SuperCollider does. `lang/sc` runs `sclang` and `scsynth` offline; `lang/compare` is the comparison.

## Tests

`go test ./...` in `dsl`, `blocks` and `lang`. Notable: `lang/live_session_test.go` plays edit sessions on the example patches; `dsl/scugens_test.go` holds numbers measured from SuperCollider; the comparison tests need SuperCollider installed and skip without it. The reference page is checked against the code (`lang/docs_test.go`).

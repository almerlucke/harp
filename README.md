# harp

A language for sound synthesis, with generators (rhythms, phrases, notes) in the language, an engine built for audio, and live editing that keeps the state of what you did not change.

It combines [psalm](../psalm) (DSP blocks, effects, synths), [genny](../genny) (generators) and [exbana](https://github.com/almerlucke/exbana) (parsing, v3.0.0 from its module), with sndfile (sound files) and go-jack. It is one Go module, built in a workspace (`go.work`) with psalm, genny, sndfile and go-jack next to it; exbana comes from its published module.

```
cd lang
go run ./cmd/harp play examples/live_demo.harp
```

Documentation is in [docs/](docs/README.md): getting started, the language, types and channels, generators, `play`, live editing, a generated function reference, the SuperCollider tools, and the architecture.

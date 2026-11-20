# harp

A language for sound synthesis, with generators (rhythms, phrases, notes) in the language, an engine built for audio, and live editing that keeps the state of what you did not change.

It combines [psalm](https://github.com/almerlucke/psalm) (DSP blocks, effects, synths), [genny](https://github.com/almerlucke/genny) (generators) and [exbana](https://github.com/almerlucke/exbana) (parsing), with sndfile (sound files). It is one Go module that builds from the tagged versions of those libraries; for developing them together, a local `go.work` can list checkouts next to it.

```
cd lang
go run ./cmd/harp play examples/live_demo.harp
```

Documentation is in [docs/](docs/README.md): getting started, the language, types and channels, generators, `play`, live editing, a generated function reference, the SuperCollider tools, and the architecture.

## License

[MIT](LICENSE).

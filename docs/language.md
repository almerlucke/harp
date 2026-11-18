# The language

A patch is a list of statements. Each is a name and an expression:

```harp
cutoff = lfo(0.3).range(300, 4000)
out = saw(110).lowpass(cutoff: cutoff).gain(0.2)
```

White space does not matter (a line that starts with a dot goes on with the line before). `//` starts a comment.

## Names with a meaning

| Name | Meaning |
|---|---|
| `out` | what is heard (required, once) |
| `bpm` | tempo, default 120 |
| `steps` | steps in a beat, default 4: the clock that generators run on |
| `seed` | seeds the random numbers: the same patch sounds the same |

`true`, `false`, `major`, `minor`, `dorian`, `pentatonic` and `pi` are always there. A name is defined once per scope.

## Values

| Written | Is |
|---|---|
| `440`, `0.5`, `1e3`, `-8` | a number |
| `c4`, `a3`, `f#2` | a note, as its MIDI number (`c4` is 60, `a3` is 57) |
| `"pad"` | a string |
| `[7 9 11]` | a list, items separated by spaces (plain items) |
| `[a * 2, b]` | a list with a comma: the items can be expressions |

Operators: `+ - * /`, with the usual precedence. A signal can be divided by a number, not by a signal.

## Calls and chains

`f(x, a, b)` and `x.f(a, b)` are the same: the thing before the dot is the first argument. So a chain reads in the order the sound flows:

```
saw(110).lowpass(cutoff: 800, res: 0.3).gain(0.5)
```

Arguments can be named (`res: 0.3`); positional ones come first. A name after a dot without parentheses is a field: `voice.pitch`, `voice.gate`.

## Your own functions

```
fn pluck(gate, note) {
  tone = saw(hz(note))
  tone.lowpass(cutoff: 2000) * env(gate, [0 1 0], [0.003 0.3], -4)
}
```

The body is statements, then the value. A function sees its parameters and the names of the patch that are defined by the time it is called, not the variables of the place it is called from. It can call functions defined before it (and itself, up to 200 deep).

## `if`

```
filter = if coin(0.5) { lowpass(x, 800) } else { highpass(x, 800) }
```

`if` is an expression. Only the branch that is taken is worked out, so nothing in the other is built or drawn. The condition must be true or false (`coin(p)` is true with chance p, decided when the patch is built).

## Errors

Mistakes are reported with the line and column, the line itself, and, where a name is nearly right, a suggestion:

```
first.harp:3:9: there is no function lowpas (did you mean lowpass?)
```

The checker also keeps signals and generators apart: see [Types and channels](types.md).

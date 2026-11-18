package blocks

import (
	"fmt"
	"sort"
	"strings"

	"harp/dsl"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// Effect hosts a module of psalm: an effect that has sockets of audio and takes its settings as messages ("rate", "depth", "mix"), as
// every effect of psalm does. The settings it has are described by the effect itself (params.Set): the host takes them from there, so
// every effect is the same few lines (effects_table.go), and what it can change is what psalm says.
//
// The sound comes in and goes out in stereo (a mono signal goes to both sides, see dsl.As). A setting can be a number (made when the patch
// is built) or a signal, which is read once in every call of the kernel (at most 64 samples) and sent as a message when it changes.
//
// Most effects work on whatever number of samples they are given. Some work on blocks of 64 (a vocoder, a shimmer): the host then gives
// them blocks of 64 on the grid of the stream, and the sound comes out one block (1.5 ms) late.

// EffectInfo says how to make an effect and what it takes
type EffectInfo struct {
	Name string
	// New makes the module with its default settings; Set describes them
	New func() psalm.Modular
	Set func() params.Set
	// Inputs is the number of channels of each socket of audio that it has: two stereo sockets (compressor: the second is the sidechain), or two mono
	// ones (a vocoder: the carrier and the modulator)
	Inputs []int
	// Blocks: it works on blocks of 64 samples (it uses the size of the buffer of psalm), see above
	Blocks bool
}

// Settings of an effect, by the key of the message
type Settings map[string]float64

// Describe is what the effect says of its settings
func (e EffectInfo) Describe() params.Set { return e.Set() }

// Effect makes an effect that plays in a patch
//
// ins has a signal for each socket of audio; a signal that is not there (the zero dsl.Sig) is a socket that is not connected: it takes
// silence, and the effect knows (a compressor with nothing in its sidechain listens to its own input)
func Effect(info EffectInfo, ins []dsl.Sig, fixed Settings, mods map[string]dsl.Sig) dsl.Sig {
	if len(ins) != len(info.Inputs) {
		panic(fmt.Sprintf("blocks: %s takes %d inputs", info.Name, len(info.Inputs)))
	}
	connected := make([]bool, len(ins))
	for k := range ins {
		connected[k] = ins[k].Channels() > 0
		if !connected[k] {
			if k == 0 {
				panic("blocks: " + info.Name + " needs its first input")
			}
			ins[k] = dsl.N(0)
		}
	}
	// the inputs of the node: the channels of each socket, then the signals of the settings that move, in the order of their keys
	var inputs []dsl.Sig
	for k, sock := range info.Inputs {
		s := dsl.As(ins[k], sock)
		for c := 0; c < sock; c++ {
			inputs = append(inputs, s.Ch(c))
		}
	}
	keys := make([]string, 0, len(mods))
	for k := range mods {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		inputs = append(inputs, dsl.Mono(mods[k]))
	}
	fixedKeys := make([]string, 0, len(fixed))
	for k := range fixed {
		fixedKeys = append(fixedKeys, k)
	}
	sort.Strings(fixedKeys)
	var id strings.Builder
	fmt.Fprintf(&id, "effect %s blocks=%v connected=%v mods=%v", info.Name, info.Blocks, connected, keys)
	for _, k := range fixedKeys {
		fmt.Fprintf(&id, " %s=%g", k, fixed[k])
	}
	nin := 0
	for _, c := range info.Inputs {
		nin += c
	}
	return dsl.CustomN(dsl.SpecN{
		Name:    info.Name,
		Key:     id.String(),
		Inputs:  inputs,
		Outputs: 2,
		New: func(rt *dsl.Rt) dsl.ProcessN {
			h := newEffectHost(rt, info, fixed, keys, connected)
			return h.process
		},
	})
}

// effectHost is one effect that plays: the module, the buffers of its sockets, and what it was last told
type effectHost struct {
	rt     *dsl.Rt
	info   EffectInfo
	m      psalm.Modular
	in     [][][]float32 // the whole buffers of the sockets of input: [socket][channel]
	out    [][]float32   // the whole buffers of the output
	keys   []string
	last   []float64
	have   []bool
	nin    int
	filled int // samples of the block of 64 that are in the input (a module that works in blocks)
	// blocks only: what the module made for the block before, that is played while the next is made
	ready  [][]float32
	played int
}

func newEffectHost(rt *dsl.Rt, info EffectInfo, fixed Settings, keys []string, connected []bool) *effectHost {
	h := &effectHost{rt: rt, info: info, keys: keys, last: make([]float64, len(keys)), have: make([]bool, len(keys))}
	inEnv(rt.SampleRate, func() {
		h.m = info.New()
		if len(fixed) > 0 {
			msg := psalm.Message{}
			for k, v := range fixed {
				msg[k] = v
			}
			h.m.ReceiveMessage(msg)
		}
	})
	for k := range info.Inputs {
		if connected[k] && k > 0 {
			h.m.Input(k).AddConnection(&psalm.Connection{}) // it listens to this socket (a sidechain)
		}
		h.nin += info.Inputs[k]
		chans := h.m.Input(k).Channels
		sock := make([][]float32, len(chans))
		for c := range chans {
			sock[c] = chans[c][:cap(chans[c])]
		}
		h.in = append(h.in, sock)
	}
	chans := h.m.Output(0).Channels
	h.out = make([][]float32, len(chans))
	for c := range chans {
		h.out[c] = chans[c][:cap(chans[c])]
	}
	if len(fixed) > 0 {
		// the settings of an effect glide to what they are set to (30 to 50 ms): the effect is run on silence for as long as they take, so that it
		// starts as it is set, and not as it is by default
		warm := int(0.4*rt.SampleRate)/dsl.BlockSize + 1
		for i := 0; i < warm; i++ {
			h.m.Synthesize()
		}
	}
	if info.Blocks {
		h.ready = make([][]float32, 2)
		for c := range h.ready {
			h.ready[c] = make([]float32, dsl.BlockSize)
		}
		h.played = dsl.BlockSize // nothing is ready yet: silence for the first block
	}
	return h
}

// tell sends the settings that moved since they were last sent
func (h *effectHost) tell(b [][]float64) {
	var msg psalm.Message
	for i, k := range h.keys {
		v := b[h.nin+i][0]
		if !h.have[i] || v != h.last[i] {
			if msg == nil {
				msg = psalm.Message{}
			}
			msg[k] = v
			h.last[i], h.have[i] = v, true
		}
	}
	if msg != nil {
		h.m.ReceiveMessage(msg)
	}
}

func (h *effectHost) process(b [][]float64, out [][]float64, n int) {
	h.tell(b)
	if h.info.Blocks {
		h.processBlocks(b, out, n)
		return
	}
	// the module works on as many samples as its buffers are long: they are made n long for this call
	at := 0
	for k := range h.in {
		for c := range h.in[k] {
			buf := h.in[k][c][:n]
			h.m.Input(k).Channels[c] = buf
			for i := 0; i < n; i++ {
				buf[i] = float32(b[at][i])
			}
			at++
		}
	}
	for c := range h.out {
		h.m.Output(0).Channels[c] = h.out[c][:n]
	}
	h.m.Synthesize()
	for c := range out {
		src := h.out[min(c, len(h.out)-1)]
		for i := 0; i < n; i++ {
			out[c][i] = float64(src[i])
		}
	}
}

func (h *effectHost) processBlocks(b [][]float64, out [][]float64, n int) {
	for i := 0; i < n; i++ {
		// what was made for the block before comes out
		if h.played < dsl.BlockSize {
			for c := range out {
				out[c][i] = float64(h.ready[min(c, len(h.ready)-1)][h.played])
			}
			h.played++
		} else {
			for c := range out {
				out[c][i] = 0
			}
		}
		at := 0
		for k := range h.in {
			for c := range h.in[k] {
				h.in[k][c][h.filled] = float32(b[at][i])
				at++
			}
		}
		h.filled++
		if h.filled == dsl.BlockSize {
			h.m.Synthesize()
			for c := range h.ready {
				copy(h.ready[c], h.out[min(c, len(h.out)-1)][:dsl.BlockSize])
			}
			h.filled, h.played = 0, 0
		}
	}
}

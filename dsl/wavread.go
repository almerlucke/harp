package dsl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
)

// ReadWAV reads a WAV file of 16, 24 or 32 bit PCM or of 32 bit float samples (what WriteWAV, WriteWAVFloat and
// SuperCollider write): the sample rate and a slice of samples for every channel, from -1 to 1 for PCM
func ReadWAV(path string) (sampleRate int, channels [][]float64, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, err
	}
	return parseWAV(data)
}

func parseWAV(d []byte) (int, [][]float64, error) {
	if len(d) < 12 || string(d[:4]) != "RIFF" || string(d[8:12]) != "WAVE" {
		return 0, nil, errors.New("not a WAV file")
	}
	var tag, nch, bits, rate int
	var pcm []byte
	haveFmt := false
	// the chunks: id, size, data (padded to an even size); the ones that are not known (LIST, PEAK, fact) are skipped
	for p := 12; p+8 <= len(d); {
		id, size := string(d[p:p+4]), int(binary.LittleEndian.Uint32(d[p+4:]))
		body := d[p+8:]
		if size > len(body) {
			size = len(body) // a file that was cut off, or a writer that does not know the size: take what is there
		}
		switch id {
		case "fmt ":
			if size < 16 {
				return 0, nil, errors.New("WAV: the fmt chunk is too short")
			}
			tag = int(binary.LittleEndian.Uint16(body[0:]))
			nch = int(binary.LittleEndian.Uint16(body[2:]))
			rate = int(binary.LittleEndian.Uint32(body[4:]))
			bits = int(binary.LittleEndian.Uint16(body[14:]))
			if tag == 0xFFFE && size >= 26 { // extensible: the real format is in the sub format
				tag = int(binary.LittleEndian.Uint16(body[24:]))
			}
			haveFmt = true
		case "data":
			pcm = body[:size]
		}
		p += 8 + size + size%2
	}
	if !haveFmt || pcm == nil {
		return 0, nil, errors.New("WAV: there is no fmt or data chunk")
	}
	if nch < 1 {
		return 0, nil, errors.New("WAV: no channels")
	}
	width := bits / 8
	if (tag == 1 && bits != 16 && bits != 24 && bits != 32) || (tag == 3 && bits != 32) || (tag != 1 && tag != 3) {
		return 0, nil, fmt.Errorf("WAV: format %d with %d bits is not supported (16, 24 or 32 bit PCM, or 32 bit float)", tag, bits)
	}
	frames := len(pcm) / (width * nch)
	out := make([][]float64, nch)
	for c := range out {
		out[c] = make([]float64, frames)
	}
	for i := 0; i < frames; i++ {
		for c := 0; c < nch; c++ {
			b := pcm[(i*nch+c)*width:]
			var v float64
			switch {
			case tag == 3:
				v = float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
			case bits == 16:
				v = float64(int16(binary.LittleEndian.Uint16(b))) / 32768
			case bits == 24:
				v = float64(int32(uint32(b[0])<<8|uint32(b[1])<<16|uint32(b[2])<<24)>>8) / 8388608
			default:
				v = float64(int32(binary.LittleEndian.Uint32(b))) / 2147483648
			}
			out[c][i] = v
		}
	}
	return rate, out, nil
}

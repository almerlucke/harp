package sc

import (
	"sync"
	"testing"
)

// renders that run at the same time do not touch each other (the score file of recordNRT is named from a counter that starts at
// the same number in every sclang: it has to be in the folder of the render), and every seed gives its own, always the same, patch
func TestRendersAtTheSameTime(t *testing.T) {
	s, err := Find("")
	if err != nil {
		t.Skip(err)
	}
	const code = "{ SinOsc.ar(exprand(100, 1000), 0, 0.1) * LFNoise1.kr(3).range(0.1, 1) ! 2 }"
	energy := func(seed int) float64 {
		x, _, err := s.RenderSeed(code, 1, 44100, seed)
		if err != nil {
			t.Error(err)
			return 0
		}
		var e float64
		for _, v := range x[0] {
			e += v * v
		}
		return e
	}
	alone := []float64{energy(1), energy(2), energy(3)}
	if alone[0] == alone[1] || alone[1] == alone[2] {
		t.Fatalf("the seeds give the same patch: %v", alone)
	}
	got := make([]float64, 3)
	var wg sync.WaitGroup
	for k := range got {
		wg.Add(1)
		go func(k int) { defer wg.Done(); got[k] = energy(k + 1) }(k)
	}
	wg.Wait()
	for k := range got {
		if got[k] != alone[k] {
			t.Errorf("seed %d: %v at the same time as others, %v alone", k+1, got[k], alone[k])
		}
	}
}

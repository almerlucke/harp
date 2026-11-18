package lang

import (
	"strconv"
	"strings"
	"testing"
)

func nums(t *testing.T, src string, n int) []float64 {
	t.Helper()
	var out []float64
	for _, f := range strings.Fields(pull(t, src, n)) {
		x, err := strconv.ParseFloat(f, 64)
		if err != nil {
			t.Fatalf("%s: %q", src, f)
		}
		out = append(out, x)
	}
	return out
}

func within(t *testing.T, src string, n int, lo, hi float64) []float64 {
	t.Helper()
	x := nums(t, src, n)
	for i, v := range x {
		if v < lo-1e-9 || v > hi+1e-9 {
			t.Errorf("%s: value %d is %v, outside %v..%v", src, i, v, lo, hi)
			break
		}
	}
	return x
}

func TestGeneratorsOfNumbersStayInTheirRange(t *testing.T) {
	within(t, `wander(50, 60, 0.3)`, 200, 50, 60)
	within(t, `wander(50, 60, 0.5, mode: "clip")`, 200, 50, 60)
	within(t, `drift(0.2, 2)`, 200, -1.2, 1.2)
	within(t, `wave(8, 10, 20)`, 64, 10, 20)
	within(t, `wave(8, 0, 1, shape: "smooth")`, 64, 0, 1)
	within(t, `orbit(3, 2, 16)`, 64, -1, 1)
	within(t, `beta(2, 5, 40, 80)`, 100, 40, 80)
	within(t, `weibull(5, 2, 1.5)`, 100, 5, 1e9)
	within(t, `pink()`, 100, 0, 1)
	within(t, `chaos("logistic")`, 100, 0, 1)
	within(t, `remap(chaos("henon"), -1.5, 1.5, 0, 10)`, 100, -2, 12)
	for _, sys := range []string{"lorenz", "rossler", "duffing", "pendulum", "standardmap"} {
		x := nums(t, `chaos("`+sys+`", axis: 1)`, 300)
		same := true
		for _, v := range x {
			if v != x[0] {
				same = false
			}
		}
		if same {
			t.Errorf("%s does not move", sys)
		}
	}
}

func TestASweepGoesFromOneEndToTheOther(t *testing.T) {
	x := nums(t, `sweep(0, 10, 11)`, 14)
	if x[0] != 0 || x[10] < 9.9999 || x[13] < 9.9999 || x[13] > 10.0001 {
		t.Errorf("%v", x)
	}
	for i := 1; i < 11; i++ {
		if x[i] < x[i-1] {
			t.Errorf("it falls at %d: %v", i, x)
		}
	}
	// a wave of a period of 4 steps comes back in 4
	w := nums(t, `wave(4, 0, 1, shape: "saw")`, 9)
	if d := w[0] - w[4]; d > 1e-9 || d < -1e-9 {
		t.Errorf("a period of 4: %v", w)
	}
}

func TestGeneratorsOfNumbersAreSeeded(t *testing.T) {
	for _, src := range []string{`wander(0, 1)`, `gauss(0, 1)`, `expo(1)`, `cauchy(0)`, `drift()`, `pink()`} {
		if a, b := pull(t, src, 20), pull(t, src, 20); a != b {
			t.Errorf("%s: another stream with the same seed", src)
		}
	}
}

func TestMistakesInGeneratorsOfNumbers(t *testing.T) {
	for src, want := range map[string]string{
		`wander(0, 1, mode: "bounse")`: "did you mean bounce",
		`wave(0)`:                      "above 0",
		`wave(4, shape: "tri")`:        "one of",
		`sweep(0, 1, 1)`:               "2 or more",
		`chaos("lorenz", axis: 3)`:     "3 values",
		`chaos("lorentz")`:             "did you mean lorenz",
		`gauss(0, 0)`:                  "above 0",
		`beta(0, 1)`:                   "above 0",
		`remap(drift(), 1, 1, 0, 1)`:   "the same",
		`drift(0.1, 0)`:                "1 to 8",
	} {
		_, err := Compile("seed = 1\nx = " + src + "\nout = sine(440).dup()\n")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v (wanted %q)", src, err, want)
		}
	}
}

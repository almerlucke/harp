package dsl

// Timeline is a signal that is a function of the time in steps (counted from the start of the patch): the shape of a song, a curve over its
// sections. key says what the function is (nodes with the same key are one node, and a live edit that does not change it keeps the node)
func Timeline(key string, f func(step float64) float64) Sig {
	return Custom(Spec{Name: "timeline", Key: key, New: func(rt *Rt) Process {
		return func(_ [][]float64, out []float64, n int) {
			pos := rt.Pos()
			for i := 0; i < n; i++ {
				out[i] = f(float64(pos+int64(i)) / rt.StepSamples)
			}
		}
	}})
}

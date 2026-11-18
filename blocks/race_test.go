//go:build race

package blocks

// the race detector slows everything down several times: deadlines mean nothing then
const raceOn = true

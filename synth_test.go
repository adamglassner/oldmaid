package main

import (
	"math"
	"testing"
	"time"
)

func peakRMS(s [2][]float64) (float64, float64) {
	p, e := 0.0, 0.0
	for _, v := range s[0] {
		p = math.Max(p, math.Abs(v))
		e += v * v
	}
	return p, math.Sqrt(e / float64(len(s[0])))
}

func TestSynth(t *testing.T) {
	r := newRoom()
	t0 := time.Now()
	m, starts := renderMusic(r)
	p, rms := peakRMS(m)
	t.Logf("music %.1fs rendered in %v, starts %v, peak %.3f rms %.4f", float64(len(m[0]))/sampleRate, time.Since(t0), starts, p, rms)
	for _, n := range sfxNames {
		t1 := time.Now()
		s := renderSfx(n, 1, r)
		p, rms := peakRMS(s)
		t.Logf("%-8s %.2fs in %v peak %.3f rms %.4f", n, float64(len(s[0]))/sampleRate, time.Since(t1), p, rms)
	}
}

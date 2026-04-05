package cmd

import (
	"math"
	"math/cmplx"
)

// fft performs an in-place radix-2 Cooley-Tukey FFT.
// Input length must be a power of 2.
func fft(x []complex128) []complex128 {
	n := len(x)
	if n <= 1 {
		return x
	}

	even := make([]complex128, n/2)
	odd := make([]complex128, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}
	even = fft(even)
	odd = fft(odd)

	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		w := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(n))) * odd[k]
		result[k] = even[k] + w
		result[k+n/2] = even[k] - w
	}
	return result
}

// magnitudeSpectrum returns magnitude of each frequency bin from real samples.
// Pads to next power of 2 before FFT.
func magnitudeSpectrum(samples []float64) []float64 {
	n := nextPow2(len(samples))
	input := make([]complex128, n)
	for i, s := range samples {
		input[i] = complex(s, 0)
	}

	spectrum := fft(input)
	mags := make([]float64, n/2)
	for i := range mags {
		mags[i] = cmplx.Abs(spectrum[i])
	}
	return mags
}

// nextPow2 returns the smallest power of 2 >= n.
func nextPow2(n int) int {
	p := 1
	for p < n {
		p *= 2
	}
	return p
}

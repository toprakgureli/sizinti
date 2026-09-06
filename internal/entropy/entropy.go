package entropy

import "math"

// Shannon returns the byte-wise Shannon entropy in bits per byte.
func Shannon(value string) float64 {
	if len(value) == 0 {
		return 0
	}
	var counts [256]int
	for i := 0; i < len(value); i++ {
		counts[value[i]]++
	}
	var result float64
	for _, count := range counts {
		if count == 0 {
			continue
		}
		probability := float64(count) / float64(len(value))
		result -= probability * math.Log2(probability)
	}
	return result
}

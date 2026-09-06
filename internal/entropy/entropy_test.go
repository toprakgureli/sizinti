package entropy

import (
	"math"
	"testing"
)

func TestShannon(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  float64
	}{
		{name: "empty"},
		{name: "repeated", value: "aaaaaaaa"},
		{name: "binary", value: "abab", want: 1},
		{name: "four symbols", value: "abcdabcd", want: 2},
		{name: "eight symbols", value: "abcdefgh", want: 3},
		{name: "unequal", value: "aaab", want: 0.8112781244591328},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Shannon(tt.value); math.Abs(got-tt.want) > 1e-12 {
				t.Errorf("Shannon() = %f, want %f", got, tt.want)
			}
		})
	}
}

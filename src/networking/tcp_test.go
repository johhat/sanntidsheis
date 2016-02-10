package tcp

import "testing"

func TestSum(t *testing.T) {
	t.Log("Testing sum function")

	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {

			actual := Sum(i, j)
			expected := i + j

			if actual != expected {
				t.Errorf("Expected sum %d but it was %d instead", expected, actual)
				return
			}
		}
	}
}

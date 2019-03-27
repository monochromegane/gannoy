package gannoy

import (
	"testing"
)

func TestHeapSortAsc(t *testing.T) {
	array := testSortArray()
	HeapSort(array, ASC, len(array))

	expects := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	for i, expect := range expects {
		if array[i].Value != expect {
			t.Errorf("Sorted array should be %v, but %v.", expects, array)
			break
		}
	}
}

func TestHeapSortAscPartial(t *testing.T) {
	array := testSortArray()
	HeapSort(array, ASC, 3)

	expects := []float64{7, 8, 9}
	for i, expect := range expects {
		if array[6:][i].Value != expect {
			t.Errorf("Sorted array should be %v, but %v.", expects, array)
			break
		}
	}
}

func TestHeapSortDesc(t *testing.T) {
	array := testSortArray()
	HeapSort(array, DESC, len(array))

	expects := []float64{9, 8, 7, 6, 5, 4, 3, 2, 1}
	for i, expect := range expects {
		if array[i].Value != expect {
			t.Errorf("Sorted array should be %v, but %v.", expects, array)
			break
		}
	}
}

func TestHeapSortDescPartial(t *testing.T) {
	array := testSortArray()
	HeapSort(array, DESC, 3)

	expects := []float64{3, 2, 1}
	for i, expect := range expects {
		if array[6:][i].Value != expect {
			t.Errorf("Sorted array should be %v, but %v.", expects, array)
			break
		}
	}
}

func testSortArray() []Sorter {
	return []Sorter{{Value: 5}, {Value: 4}, {Value: 9}, {Value: 2}, {Value: 1}, {Value: 8}, {Value: 7}, {Value: 6}, {Value: 3}}
}

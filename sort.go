package gannoy

func HeapSort(array []float64, order, last int) {
	var heapifier heapifier
	switch order {
	case ASC:
		heapifier = heapifyFunc(downHeapify)
	default:
		heapifier = heapifyFunc(upHeapify)
	}

	heapSort(heapifier, array, last)
}

type heapifier interface {
	heapify([]float64, int, int)
}

type heapifyFunc func([]float64, int, int)

func (f heapifyFunc) heapify(array []float64, root, length int) {
	f(array, root, length)
}

func heapSort(heapifier heapifier, array []float64, last int) {
	// initialize
	for i := len(array) / 2; i >= 0; i-- {
		heapifier.heapify(array, i, len(array))
	}

	// remove top and do heapify
	bp := len(array) - last
	for length := len(array); length > 1; length-- {
		lastIndex := length - 1
		array[0], array[lastIndex] = array[lastIndex], array[0]
		heapifier.heapify(array, 0, lastIndex)
		if lastIndex == bp {
			break
		}
	}
}

func downHeapify(array []float64, root, length int) {
	max := root
	l := (root * 2) + 1
	r := l + 1

	if l < length && array[l] > array[max] {
		max = l
	}

	if r < length && array[r] > array[max] {
		max = r
	}

	if max != root {
		array[root], array[max] = array[max], array[root]
		downHeapify(array, max, length)
	}
}

func upHeapify(array []float64, root, length int) {
	min := root
	l := (root * 2) + 1
	r := l + 1

	if l < length && array[l] < array[min] {
		min = l
	}

	if r < length && array[r] < array[min] {
		min = r
	}

	if min != root {
		array[root], array[min] = array[min], array[root]
		upHeapify(array, min, length)
	}
}

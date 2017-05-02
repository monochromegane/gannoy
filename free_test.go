package gannoy

import "testing"

func TestFreePopEmpty(t *testing.T) {
	free := newFree()
	id, err := free.pop()
	if id != 0 {
		t.Errorf("Free pop with empty list should return 0, but %d.", id)
	}
	if err == nil {
		t.Errorf("Free pop with empty list should return error.")
	}
}

func TestFreePop(t *testing.T) {
	free := newFree()
	free.push(1)
	free.push(2)

	id, err := free.pop()
	if id != 2 {
		t.Errorf("Free pop should return 2, but %d.", id)
	}
	if err != nil {
		t.Errorf("Free pop should not return error.")
	}
}

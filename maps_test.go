package gannoy

import "testing"

func TestMapsGetIdNotFound(t *testing.T) {
	maps := newMaps()

	id, err := maps.getId(0)
	if id != -1 {
		t.Errorf("Maps getId when not found should return -1, but %d.", id)
	}
	if err == nil {
		t.Errorf("Maps getId when not found should return error.")
	}
}

func TestMapsGetId(t *testing.T) {
	maps := newMaps()
	maps.add(1, 10)
	id, err := maps.getId(10)
	if id != 1 {
		t.Errorf("Maps getId should return 1, but %d.", id)
	}
	if err != nil {
		t.Errorf("Maps getId should not return error.")
	}

	maps.remove(10)
	id, err = maps.getId(10)
	if id != -1 {
		t.Errorf("Maps getId when not found should return -1, but %d.", id)
	}
	if err == nil {
		t.Errorf("Maps getId when not found should return error.")
	}
}

func TestMapsIsExist(t *testing.T) {
	maps := newMaps()
	if maps.isExist(10) {
		t.Errorf("Maps isExist when not exist should return false.")
	}

	maps.add(1, 10)

	if !maps.isExist(10) {
		t.Errorf("Maps isExist when exist should return true.")
	}
}

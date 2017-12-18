package gannoy

import (
	"os"
	"testing"
	"time"
)

func TestBinLogOpen(t *testing.T) {
	path := tempDatabaseDir("db")
	defer os.Remove(path)

	bin := NewBinLog(path)
	err := bin.Open()
	if err != nil {
		t.Errorf("BinLog.Open should not return error, but return %v", err)
	}
	defer bin.Close()
}

func TestBinLogAdd(t *testing.T) {
	path := tempDatabaseDir("db")
	defer os.Remove(path)

	bin := NewBinLog(path)
	err := bin.Open()
	if err != nil {
		t.Errorf("BinLog.Open should not return error, but return %v", err)
	}
	defer bin.Close()

	expectKey := 1
	expectAction := UPDATE
	expectFeature := []byte("{features:[0.0,0.0,0.0]}")

	err = bin.Add(expectKey, expectAction, expectFeature)
	if err != nil {
		t.Errorf("BinLog.Add should not return error, but return %v", err)
	}
}

func TestBinLogGetAndCount(t *testing.T) {
	path := tempDatabaseDir("db")
	defer os.Remove(path)

	bin := NewBinLog(path)
	err := bin.Open()
	if err != nil {
		t.Errorf("BinLog.Open should not return error, but return %v", err)
	}
	defer bin.Close()

	expectKey := 1
	expectAction := UPDATE
	expectFeature := []byte("{features:[0.0,0.0,0.0]}")

	err = bin.Add(expectKey, expectAction, expectFeature)
	if err != nil {
		t.Errorf("BinLog.Add should not return error, but return %v", err)
	}

	current := time.Now().Format("2006-01-02 15:04:05")

	cnt, err := bin.Count(current)
	if err != nil {
		t.Errorf("BinLog.Count should not return error, but return %v", err)
	}
	if cnt != 1 {
		t.Errorf("BinLog.Count should return 1, but return %d", cnt)
	}

	rows, err := bin.Get(current)
	if err != nil {
		t.Errorf("BinLog.Get should not return error, but return %v", err)
	}

	for rows.Next() {
		var key int
		var action int
		var features []byte

		rows.Scan(&key, &action, &features)

		if key != expectKey {
			t.Errorf("BinLog.Get should return %d as key, but return %d", expectKey, key)
		}
		if action != expectAction {
			t.Errorf("BinLog.Get should return %d as action, but return %d", expectAction, action)
		}
		sf := string(features)
		esf := string(expectFeature)
		if sf != esf {
			t.Errorf("BinLog.Get should return %s as features, but return %s", esf, sf)
		}
	}
}

func TestBinLogClear(t *testing.T) {
	path := tempDatabaseDir("db")
	defer os.Remove(path)

	bin := NewBinLog(path)
	err := bin.Open()
	if err != nil {
		t.Errorf("BinLog.Open should not return error, but return %v", err)
	}
	defer bin.Close()

	expectKey := 1
	expectAction := UPDATE
	expectFeature := []byte("{features:[0.0,0.0,0.0]}")

	err = bin.Add(expectKey, expectAction, expectFeature)
	if err != nil {
		t.Errorf("BinLog.Add should not return error, but return %v", err)
	}

	current := time.Now().Format("2006-01-02 15:04:05")

	cnt, err := bin.Count(current)
	if err != nil {
		t.Errorf("BinLog.Count should not return error, but return %v", err)
	}
	if cnt != 1 {
		t.Errorf("BinLog.Count should return 1, but return %d", cnt)
	}

	err = bin.Clear(current)
	if err != nil {
		t.Errorf("BinLog.Clear should not return error, but return %v", err)
	}

	cnt, err = bin.Count(current)
	if err != nil {
		t.Errorf("BinLog.Count should not return error, but return %v", err)
	}
	if cnt != 0 {
		t.Errorf("BinLog.Count should return 0, but return %d", cnt)
	}
}

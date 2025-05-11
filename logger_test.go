package main

import (
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	dt, err := time.Parse("2006-01-02T15-04-05", "2024-02-01T03-04-05")
	if err != nil {
		t.Fatalf("time Parse failed: %s", err.Error())
	}

	l := newLogfile("", "", true)
	AssertEquals(t, l.formatName(dt, l.name), "")
	AssertEquals(t, l.formatName(dt, []byte("%y-%m-%dT%H-%M-%S")), "2024-02-01T03-04-05")
	
	dt, err = time.Parse("2006-01-02T15-04-05", "1984-11-20T23-59-59")
	if err != nil {
		t.Fatalf("time Parse failed: %s", err.Error())
	}
	AssertEquals(t, l.formatName(dt, []byte("%y-%m-%dT%H-%M-%S")), "1984-11-20T23-59-59")
	AssertEquals(t, l.formatName(dt, []byte("%%y-%m-%dT%H-%M-%S%")), "%1984-11-20T23-59-59%")
	AssertEquals(t, l.formatName(dt, []byte("%%y%m%d%H%M%S%")), "%19841120235959%")
	AssertEquals(t, l.formatName(dt, []byte("log-%y%m%d.txt")), "log-19841120.txt")

	// b := []byte("%y%m%d%H%M%S")
	// t1 := time.Now().UnixMilli()
	// for i := 0; i < 500000; i++ {
	// 	l.formatName(dt, b)
	// }
	// t2 := time.Now().UnixMilli() - t1
	// fmt.Printf("Time : %d ms\n", t2)

}

func AssertEquals(t *testing.T, s string, expected string) {
	if s != expected {
		t.Fatalf("AssertEquals Error:\nValue   %s\nExpected:%s", s, expected)
	}
}

package main

import "testing"

func TestFormat(t *testing.T) {
	/*
		echo fefe1102000101022444fc0400010000fb000477|go run main.go
		0xfe 0xfe 0x11 0x02 0x00 0x01 0x01 0x02 0x24 0x44 0xfc 0x04 0x00 0x01 0x00 0x00 0xfb 0x00 0x04 0x77
	*/

	var input = []byte("fefe1102000101022444fc0400010000fb000477")
	var expected = "0xfe 0xfe 0x11 0x02 0x00 0x01 0x01 0x02 0x24 0x44 0xfc 0x04 0x00 0x01 0x00 0x00 0xfb 0x00 0x04 0x77"
	result := Format(input)
	if result != expected {
		t.Fatal("no", result, expected)
	}
}

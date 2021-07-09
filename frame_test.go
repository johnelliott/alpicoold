package main

import (
	"fmt"
	"testing"
)

func TestFrame(t *testing.T) {
	f := Frame{On: 1}
	fmt.Println("f", f)
	j, err := f.MarshalJSON()
	if err != nil {
		t.Fatal("failed")
	}
	fmt.Println("j", j)
}

package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"os"
)

func main() {

	scanner := bufio.NewScanner(os.Stdin)
	// Scan throgh lines in file
	for scanner.Scan() {
		src := scanner.Bytes()
		fmt.Println(Format(src))

	}
}

func Format(src []byte) string {
	dst := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(dst, src)
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("% 0#x", dst[:n])
}

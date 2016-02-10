package main

// #include "elev.h"
// #cgo CFLAGS: -std=c11 -g -Wall -Wextra
// #cgo LDFLAGS: -L. -lcomedi -lm

import (
	"C"
	"fmt"
)

func main() {
	fmt.Println("Hello there")
	init()
}

func init() bool {
	C.elev_init()

	return false
}

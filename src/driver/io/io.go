package io

/*
	#cgo CFLAGS: -std=c11
	#cgo LDFLAGS: -lcomedi -lm
	#include "channels.h"
	#include "io.h"
*/
import "C"

import (
	"errors"
	"sync"
)

const (
	InitFailureCode = 0
)

var once sync.Once

func Init() error {

	var err error

	err = errors.New("Elevator HW allready initialized.")

	once.Do(func() {

		status := C.io_init()

		if status == InitFailureCode {
			err = errors.New("Init of elevator IO failed.")
		} else {
			err = nil
		}
	})

	return err
}

func SetBit(channel int) {
	C.io_set_bit(C.int(channel))
}

func ClearBit(channel int) {
	C.io_clear_bit(C.int(channel))
}

func WriteAnalog(channel, value int) {
	C.io_write_analog(C.int(channel), C.int(value))
}

func ReadBit(channel int) int {
	return int(C.io_read_bit(C.int(channel)))
}

func ReadAnalog(channel int) int {
	return int(C.io_read_analog(C.int(channel)))
}

package main

/*
#include <termios.h>
#include <unistd.h>
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"
)

type Termios C.struct_termios

func enableRawMode(fd int) *Termios {
	var oldState Termios
	C.tcgetattr(C.int(fd), (*C.struct_termios)(unsafe.Pointer(&oldState)))

	newState := oldState
	newState.c_lflag &^= (C.ECHO | C.ICANON)
	C.tcsetattr(C.int(fd), C.TCSANOW, (*C.struct_termios)(unsafe.Pointer(&newState)))
	return &oldState
}
func restoreMode(fd int, state *Termios) {
	C.tcsetattr(C.int(fd), C.TCSANOW, (*C.struct_termios)(unsafe.Pointer(state)))
}

package main

// #cgo pkg-config: GraphicsMagickWand
// #include <stdlib.h>
// #include <stdio.h>
// #include <string.h>
// #include <wand/magick_wand.h>
import "C"
import (
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/sys/unix"
)

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/seccomp.h#L11
const seccompModeStrict = 1

func strictMode() {
	log("Setting strict mode")
	err := unix.Prctl(unix.PR_SET_SECCOMP, seccompModeStrict, 0, 0, 0)
	if err != nil {
		log(err)
	}
	log("Strict mode set")
}

func main() {
	args := os.Args
	_, err := C.InitializeMagick(C.CString(args[0]))
	if err != nil {
		log("Failed initializing GraphicsMagick:", err)
	}

	wand, err := C.NewMagickWand()
	if err != nil {
		fail("Failed obtaining MagickWand:", err)
	}

	imageData, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fail("Failed reading source image:", err)
	}

	// NOT WORKING
	// strictMode()

	log("MagickReadImageBlob")
	magickOp("MagickReadImageBlob", C.MagickReadImageBlob(wand, (*C.uchar)(&imageData[0]), C.ulong(len(imageData))))
	log("MagickResizeImage")
	magickOp("MagickResizeImage", C.MagickResizeImage(wand, 200, 200, C.LanczosFilter, 0.0))
	log("MagickWriteImage")
	magickOp("MagickWriteImage", C.MagickWriteImage(wand, C.CString("-")))
}

func magickOp(opn string, status C.uint) {
	switch status {
	case C.MagickPass:
		log(opn, "- success")
	case C.MagickFail:
		fail(opn, "- fail")
	default:
		fail(opn, "- unexpected status:", status)
	}
}

func log(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

package main

// #cgo pkg-config: GraphicsMagickWand
// #include <stdlib.h>
// #include <stdio.h>
// #include <string.h>
// #include <wand/magick_wand.h>
import "C"
import (
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/seccomp.h#L11
const seccompModeStrict = 1

func strictMode() {
	println("Setting strict mode")
	err := unix.Prctl(unix.PR_SET_SECCOMP, seccompModeStrict, 0, 0, 0)
	if err != nil {
		log.Fatalln(err)
	}
	println("Strict mode set")
}

func main() {
	args := os.Args
	_, err := C.InitializeMagick(C.CString(args[0]))
	if err != nil {
		log.Fatalln("Failed initializing GraphicsMagick:", err)
	}

	wand, err := C.NewMagickWand()
	if err != nil {
		log.Fatalln("Failed obtaining MagickWand:", err)
	}

	imageData, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalln("Failed reading source image:", err)
	}

	magickOp("MagickReadImageBlob", C.MagickReadImageBlob(wand, (*C.uchar)(&imageData[0]), C.ulong(len(imageData))))
	magickOp("MagickResizeImage", C.MagickResizeImage(wand, 200, 200, C.LanczosFilter, 0.0))
	magickOp("MagickWriteImage", C.MagickWriteImage(wand, C.CString("-")))
}

func magickOp(opn string, status C.uint) {
	switch status {
	case C.MagickPass:
		log.Println(opn, "- success")
	case C.MagickFail:
		log.Fatalln(opn, "- fail")
	default:
		log.Fatalln(opn, "- unexpected status:", status)
	}
}

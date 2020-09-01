package main

/*
#cgo pkg-config: GraphicsMagickWand

#include <stdio.h>
#include <wand/magick_wand.h>

void printMagickError(MagickWand *wand) {
	char *err;
	ExceptionType errType;

	err = MagickGetException(wand, &errType);
	fprintf(stderr, "MagickError %d - %s\n", errType, err);

	MagickRelinquishMemory(err);
}
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

var allowedSyscalls = []string{
	// REQUIRED
	"brk",
	"write",
	"futex",
	"rt_sigprocmask",
	"sigaltstack",
	"exit_group",
	// OBSERVED
	// "mmap",
	// "munmap",
	// "mprotect",
	// "prlimit64",
	// "fstat",
	// "access",
	// "openat",
	// "close",
	// "read",
	// "pread64",
	// "lseek",
	// "getdents64",
	// "readlinkat",
	// "fcntl",
	// "gettid",
	// "sched_getaffinity",
	// "times",
	// "set_tid_address",
	// "mlock",
	// "set_robust_list",
	// "arch_prctl",
	// "clone",
	// "rt_sigaction",
	// "rt_sigreturn",
	// "sysinfo",
	// "uname",
}

func main() {
	widthParam := os.Getenv("GL_RESIZE_IMAGE_WIDTH")
	requestedWidth, err := strconv.Atoi(widthParam)
	if err != nil {
		fail("Failed parsing GL_RESIZE_IMAGE_WIDTH; not a valid integer:", widthParam)
	}

	args := os.Args
	_, err = C.InitializeMagick(C.CString(args[0]))
	if err != nil {
		fail("Failed initializing GraphicsMagick:", err)
	}

	wand, err := C.NewMagickWand()
	if err != nil {
		fail("Failed obtaining MagickWand:", err)
	}
	defer C.DestroyMagickWand(wand)

	imageData, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fail("Failed reading image data from stdin", err)
	}

	withSeccomp(func() {
		runMagick("read_image", wand, readImage(imageData))
		runMagick("scale_image", wand, scaleImage(requestedWidth))
		runMagick("write_image", wand, writeImage)
	})
}

func runMagick(opName string, wand *C.MagickWand, op func(*C.MagickWand) C.uint) {
	if op(wand) == C.MagickFail {
		C.printMagickError(wand)
		fail(opName, "failed")
	}
}

func readImage(imageData []byte) func(*C.MagickWand) C.uint {
	return func(wand *C.MagickWand) C.uint {
		return C.MagickReadImageBlob(wand, (*C.uchar)(&imageData[0]), C.ulong(len(imageData)))
	}
}

func writeImage(wand *C.MagickWand) C.uint {
	defer C.fflush(C.stdout)
	return C.MagickWriteImageFile(wand, C.stdout)
}

func scaleImage(requestedWidth int) func(*C.MagickWand) C.uint {
	return func(wand *C.MagickWand) C.uint {
		currentWidth := C.MagickGetImageWidth(wand)
		currentHeight := C.MagickGetImageHeight(wand)
		aspect := C.float(currentHeight) / C.float(currentWidth)
		newWidth := C.float(requestedWidth)
		newHeight := aspect * newWidth

		return C.MagickScaleImage(wand, C.ulong(newWidth), C.ulong(newHeight))
	}
}

func log(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

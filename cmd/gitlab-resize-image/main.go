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
	"os"
	"strconv"
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
)

var allowedSyscalls = []string{
	// memory & resource management
	"mmap",
	"munmap",
	"mprotect",
	"brk",
	"prlimit64",
	// file & directory management
	"fstat",
	"access",
	"openat",
	"close",
	"read",
	"pread64",
	"write",
	"lseek",
	"getdents64",
	"readlinkat",
	"fcntl",
	// thread management
	"gettid",
	"set_tid_address",
	"mlock",
	"futex",
	"sched_getaffinity",
	"set_robust_list",
	"arch_prctl",
	// process & signal management
	"clone",
	"times",
	"exit_group",
	"rt_sigaction",
	"rt_sigprocmask",
	"rt_sigreturn",
	"sigaltstack",
	// other
	"sysinfo",
	"uname",
}

func enterSeccompMode() {
	// create a "reject all" filter that always returns "Operation not permitted"
	filter, err := seccomp.NewFilter(seccomp.ActErrno.SetReturnCode(int16(syscall.EPERM)))
	if err != nil {
		fail(err)
	}
	// allow only syscalls in the given list
	for _, syscall := range allowedSyscalls {
		id, err := seccomp.GetSyscallFromName(syscall)
		if err != nil {
			fail(err)
		}
		filter.AddRule(id, seccomp.ActAllow)
	}
	filter.Load()
}

func main() {
	enterSeccompMode()

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

	runMagick("read_image", wand, readImage)
	runMagick("scale_image", wand, scaleImage(requestedWidth))
	runMagick("write_image", wand, writeImage)
}

func runMagick(opName string, wand *C.MagickWand, op func(*C.MagickWand) C.uint) {
	if op(wand) == C.MagickFail {
		C.printMagickError(wand)
		fail(opName, "failed")
	}
}

func readImage(wand *C.MagickWand) C.uint {
	return C.MagickReadImage(wand, C.CString("-"))
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

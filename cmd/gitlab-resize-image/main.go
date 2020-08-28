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
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
)

var allowedSyscalls = []string{
	"clone",
	"rt_sigaction",
	"mmap",
	"mprotect",
	"read",
	"rt_sigprocmask",
	"openat",
	"mlock",
	"futex",
	"close",
	"fstat",
	"times",
	"getdents64",
	"readlinkat",
	"brk",
	"sched_getaffinity",
	"fcntl",
	"munmap",
	"prlimit64",
	"write",
	"lseek",
	"sigaltstack",
	"sysinfo",
	"rt_sigreturn",
	"uname",
	"arch_prctl",
	"set_robust_list",
	"gettid",
	"set_tid_address",
	"pread64",
	"access",
}

func enterSeccompMode() {
	log("Entering seccomp mode")
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
	log("Seccomp mode set")
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

	enterSeccompMode()

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

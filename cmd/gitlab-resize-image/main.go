package main

/*
#cgo pkg-config: GraphicsMagickWand

#include <wand/magick_wand.h>
*/
import "C"
import (
	"fmt"
	"io/ioutil"
	"os"
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

	magickOp("MagickReadImageBlob", C.MagickReadImageBlob(wand, (*C.uchar)(&imageData[0]), C.ulong(len(imageData))))
	magickOp("MagickResizeImage", C.MagickResizeImage(wand, 200, 200, C.LanczosFilter, 0.0))
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

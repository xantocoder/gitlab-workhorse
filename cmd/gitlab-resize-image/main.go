package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/h2non/bimg"

	seccomp "github.com/seccomp/libseccomp-golang"
)

var allowedSyscalls = []string{
	"futex",
	"mmap",
	"openat",
	"read",
	"seccomp",
	"fstat",
	"close",
	"mprotect",
	"getdents64",
	"pread64",
	"brk",
	"write",
	"prctl",
	"arch_prctl",
	"stat",
	"munmap",
	"rt_sigaction",
	"rt_sigprocmask",
	"rt_sigreturn",
	"access",
	"getpid",
	"clone",
	"uname",
	"fcntl",
	"ftruncate",
	"unlink",
	"umask",
	"sigaltstack",
	"statfs",
	"mlock",
	"gettid",
	"sched_getaffinity",
	"set_tid_address",
	"readlinkat",
	"set_robust_list",
	"prlimit64",
	"getrandom",
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
	imageData, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fail("Failed reading source image:", err)
	}

	enterSeccompMode()

	log("Resizing image...")
	resizedImage, err := bimg.NewImage(imageData).Resize(200, 200)
	if err != nil {
		fail("Failed resizing source image:", err)
	}

	log("Done! Writing to stdout...")
	bytesWritten, err := os.Stdout.Write(resizedImage)
	if err != nil {
		fail("Failed writing to stdout:", err)
	}
	if bytesWritten < len(resizedImage) {
		fail("Failed writing image data:", bytesWritten, "/", len(resizedImage), "bytes written")
	}
	log("All done!")
}

func log(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

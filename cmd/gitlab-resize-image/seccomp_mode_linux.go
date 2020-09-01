package main

import (
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
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

func withSeccomp(fn func()) {
	// create a "reject all" filter that always returns "Operation not permitted"
	filter, err := seccomp.NewFilter(seccomp.ActErrno.SetReturnCode(int16(syscall.EPERM)))
	if err != nil {
		fail(err)
	}
	filter.SetNoNewPrivsBit(true)
	// allow only syscalls in the given list
	for _, syscall := range allowedSyscalls {
		id, err := seccomp.GetSyscallFromName(syscall)
		if err != nil {
			fail(err)
		}
		filter.AddRule(id, seccomp.ActAllow)
	}
	filter.Load()
	defer filter.Release()

	fn()
}

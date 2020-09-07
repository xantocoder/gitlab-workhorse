package main

import (
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
)

var allowedSyscalls = []string{
	// REQUIRED
	"read",
	"futex",
	"write",
	"mmap",
	"exit_group",
	"mprotect",
	"clone",

	"sigaltstack",
	"rt_sigprocmask",
	// ALL OBSERVED
	//"rt_sigaction",
	//"mmap",
	//"nanosleep",
	//"clock_gettime",
	//"futex",
	//"clone",
	//"read",
	//"rt_sigprocmask",
	//"mprotect",
	//"readlinkat",
	//"openat",
	//"close",
	//"fcntl",
	//"sigaltstack",
	//"fstat",
	//"brk",
	//"munmap",
	//"access",
	//"sched_getaffinity",
	//"set_robust_list",
	//"prlimit64",
	//"gettid",
	//"arch_prctl",
	//"set_tid_address",
	//"execve",
	//"write",
	//"access",
	//"exit_group",
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

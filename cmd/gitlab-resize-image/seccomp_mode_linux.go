package main

import (
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
)

var allowedSyscalls = []string{
    "read",
    "futex",
    "write",
    "mmap",
    "exit_group",
    "mprotect",
    "clone",
    "sigaltstack",
    "rt_sigprocmask",
//     # OPTIONAL
//     "close",
//     "nanosleep",
//     "munmap",
//     "brk",
//     "rt_sigreturn",
//     "access",
//     "rt_sigaction",
//     "execve",
//     "fcntl",
//     "arch_prctl",
//     "gettid",
//     "fstat",
//     "sched_getaffinity",
//     "getdents64",
//     "set_tid_address",
//     "openat",
//     "readlinkat",
//     "set_robust_list",
//     "prlimit64",
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

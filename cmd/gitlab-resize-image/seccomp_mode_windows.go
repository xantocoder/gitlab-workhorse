package main

func withSeccomp(fn func()) { fn() } // seccomp is available in Linux kernel only

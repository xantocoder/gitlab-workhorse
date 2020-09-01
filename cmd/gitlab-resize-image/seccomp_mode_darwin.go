package main

func withSeccomp(fn func()) {} // seccomp is available in Linux kernel only

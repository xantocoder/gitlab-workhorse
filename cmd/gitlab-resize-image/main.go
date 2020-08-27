package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/sys/unix"

	"github.com/h2non/bimg"
)

// https://github.com/torvalds/linux/blob/master/include/uapi/linux/seccomp.h#L11
const seccompModeStrict = 1

func strictMode() {
	log("Setting strict mode")
	err := unix.Prctl(unix.PR_SET_SECCOMP, seccompModeStrict, 0, 0, 0)
	if err != nil {
		log(err)
	}
	log("Strict mode set")
}

func main() {
	imageData, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fail("Failed reading source image:", err)
	}

	// NOT WORKING
	// strictMode()

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

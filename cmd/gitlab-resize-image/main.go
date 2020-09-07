package main

import (
	"bytes"
	"fmt"
	"github.com/anthonynsimon/bild/transform"
	"image/png"
	"io/ioutil"
	"os"
)

func main() {
	imageData, _ := ioutil.ReadAll(os.Stdin)
	withSeccomp(func() {
		img, _ := png.Decode(bytes.NewBuffer(imageData))
		resized := transform.Resize(img, 40, 40, transform.Linear)
		_ = png.Encode(os.Stdout, resized)
	})
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

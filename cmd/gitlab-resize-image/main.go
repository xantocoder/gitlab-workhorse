package main

import (
	"bytes"
	"fmt"
	"github.com/anthonynsimon/bild/transform"
	"image/jpeg"
	"io/ioutil"
	"os"
)

func main() {
	imageData, _ := ioutil.ReadAll(os.Stdin)
	withSeccomp(func() {
		img, _ := jpeg.Decode(bytes.NewBuffer(imageData))
		resized := transform.Resize(img, 40, 40, transform.Linear)
		_ = jpeg.Encode(os.Stdout, resized, nil)
	})
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

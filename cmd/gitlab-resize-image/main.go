package main

import "C"
import (
	"fmt"
	"os"
	"io"
	"strconv"
	"image"
	"github.com/disintegration/imaging"
)

func main() {
 	widthParam := os.Getenv("GL_RESIZE_IMAGE_WIDTH")
 	requestedWidth, err := strconv.Atoi(widthParam)
 	if err != nil {
 		fail("Failed parsing GL_RESIZE_IMAGE_WIDTH; not a valid integer:", widthParam)
 	}

 	withSeccomp(func() {
        var reader io.Reader = os.Stdin
        src, extension, err := image.Decode(reader)

        if err != nil {
            fail("failed to open image: %v", err)
        }

   	    format, err := imaging.FormatFromExtension(extension)

 	    if err != nil {
             fail("failed to find extension: %v", err)
        }

	    image := imaging.Resize(src, requestedWidth, 0, imaging.Lanczos)
	    imaging.Encode(os.Stdout, image, format)
 	})
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

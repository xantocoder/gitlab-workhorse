package main

import (
	"fmt"
	"image"
	"os"
	"strconv"

	"github.com/disintegration/imaging"
)

var allowedFormats = [3]string{"png", "jpg", "jpeg"}

func main() {
	if err := _main(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: fatal: %v\n", os.Args[0], err)
		os.Exit(1)
	}
}

func _main() error {
	widthParam := os.Getenv("GL_RESIZE_IMAGE_WIDTH")
	requestedWidth, err := strconv.Atoi(widthParam)
	if err != nil {
		return fmt.Errorf("GL_RESIZE_IMAGE_WIDTH: %w", err)
	}

	src, formatName, err := image.Decode(os.Stdin)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	if !isAllowedFormat(formatName) {
		return fmt.Errorf("format is prohibited: %s", formatName)
	}

	imagingFormat, err := imaging.FormatFromExtension(formatName)
	if err != nil {
		return fmt.Errorf("find imaging format: %w", err)
	}

	image := imaging.Resize(src, requestedWidth, 0, imaging.Lanczos)
	return imaging.Encode(os.Stdout, image, imagingFormat)
}

func isAllowedFormat(f string) bool {
	for _, x := range allowedFormats {
		if f == x {
			return true
		}
	}
	return false
}

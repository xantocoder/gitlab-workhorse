package main

import (
	"fmt"
	"strconv"
	"os"
	"bytes"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/imageresizer"
)

func main() {
	imageURL := os.Getenv("WH_RESIZE_IMAGE_URL")
	requestedWidth, err := strconv.Atoi(os.Getenv("WH_RESIZE_IMAGE_WIDTH"))

	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed reading image width:", err)
	}

	fmt.Fprintln(os.Stderr, "Downloading image data ...")
	imageData, err := imageresizer.ReadAllData(imageURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed loading image data:", err)
	}

	fmt.Fprintln(os.Stderr, "imageURL:", imageURL)
	fmt.Fprintln(os.Stderr, "width:", requestedWidth)

	resizedImageData, _, err := imageresizer.ResizeImage(imageData, uint(requestedWidth), "")

	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed resizing image:", err)
	}

	//TODO: this can probably be made more efficient by write multiple chunks
	// instead of buffering upfront, then writing it all at once
	var buffer bytes.Buffer
	buffer.Write(resizedImageData)
	buffer.WriteTo(os.Stdout)	
}
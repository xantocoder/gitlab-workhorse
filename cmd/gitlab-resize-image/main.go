package main

import (
	"strconv"
	"os"
	"bytes"
	"log"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/imageresizer"
)

func main() {
	imageURL := os.Getenv("WH_RESIZE_IMAGE_URL")
	requestedWidth, err := strconv.Atoi(os.Getenv("WH_RESIZE_IMAGE_WIDTH"))

	if err != nil {
		log.Fatalln("Failed reading image width:", err)
	}

	imageData, err := imageresizer.ReadAllData(imageURL)
	if err != nil {
		log.Fatalln("Failed downloading image data:", err)
	}

	resizedImageData, _, err := imageresizer.ResizeImage(imageData, uint(requestedWidth), "")

	if err != nil {
		log.Fatalln("Failed resizing image:", err)
	}

	//TODO: this can probably be made more efficient by write multiple chunks
	// instead of buffering upfront, then writing it all at once
	var buffer bytes.Buffer
	buffer.Write(resizedImageData)
	buffer.WriteTo(os.Stdout)	
}
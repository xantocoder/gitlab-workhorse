package main

import (
	"strconv"
	"os"
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

	bytesWritten, err := os.Stdout.Write(resizedImageData)
	if err != nil {
		log.Fatalln("Failed writing image data to stdout: ", err)
	}

	if bytesWritten != len(resizedImageData) {
		log.Fatalf("Failed writing all image data (written bytes: %d, image bytes: %d)", bytesWritten, len(resizedImageData))
	}
}
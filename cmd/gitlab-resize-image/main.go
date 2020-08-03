package main

import (
	"strconv"
	"os"
	"os/exec"
	"log"
	"fmt"
	"bytes"

	"gitlab.com/gitlab-org/gitlab-workhorse/internal/imageresizer"
)

func main() {
	resizeStrategy := os.Getenv("WH_RESIZE_STRATEGY")
	if resizeStrategy == "" {
		log.Fatalln("Must provide WH_RESIZE_STRATEGY=[bimg|gmagick]")
	}

	imageURL := os.Getenv("WH_RESIZE_IMAGE_URL")
	requestedWidth, err := strconv.Atoi(os.Getenv("WH_RESIZE_IMAGE_WIDTH"))

	if err != nil {
		log.Fatalln("Failed reading image width:", err)
	}

	imageData, err := imageresizer.ReadAllData(imageURL)
	if err != nil {
		log.Fatalln("Failed downloading image data:", err)
	}

	var resizedImageData []byte
	if resizeStrategy == "bimg"{
		resizedImageData, _, err = imageresizer.ResizeImage(imageData, uint(requestedWidth), "")
	} else {
		resizedImageData, err = resizeImageGMagick(imageData, requestedWidth)
	}

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

func resizeImageGMagick(imageData []byte, width int) ([]byte, error) {
	cmd := exec.Command("gm", "convert", "-resize", fmt.Sprintf("%dx", width), "-", "-")
	var inBuffer bytes.Buffer
	cmd.Stdin = &inBuffer
	cmd.Stderr = os.Stderr

	inBuffer.Write(imageData)

	outData, err := cmd.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	return outData, nil
}
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
	location := os.Getenv("WH_RESIZE_IMAGE_LOCATION")
	requestedWidth, err := strconv.Atoi(os.Getenv("WH_RESIZE_IMAGE_WIDTH"))

	if err != nil {
		log.Fatalln("Failed reading image width:", err)
	}

	imageData, err := imageresizer.ReadAllData(location)
	if err != nil {
		log.Fatalln("Failed downloading image data:", err)
	}

	if err := resizeImageGMagick(imageData, requestedWidth); err != nil {
		log.Fatalln("Failed resizing image:", err)
	}
}

func resizeImageGMagick(imageData []byte, width int) error {
	cmd := exec.Command("gm", "convert", "-resize", fmt.Sprintf("%dx", width), "-", "-")
	var inBuffer bytes.Buffer
	cmd.Stdin = &inBuffer
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	inBuffer.Write(imageData)

	err := cmd.Start()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
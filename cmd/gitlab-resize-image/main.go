package main

import (
	"fmt"
	"strconv"
	"os"
)

func main() {
	imageURL := os.Getenv("WH_RESIZE_IMAGE_URL")
	requestedWidth, err := strconv.Atoi(os.Getenv("WH_RESIZE_IMAGE_WIDTH"))

	if err != nil {
		fmt.Println("Failed reading image width:", err)
	}

	fmt.Printf("URL = %s / width = %d", imageURL, requestedWidth)
}
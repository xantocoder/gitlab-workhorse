package imageresizer

import (
	"fmt"
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"time"

	"github.com/nfnt/resize"
)

type ImageFormat int

const (
	ImageFormatUnknown ImageFormat = iota
	ImageFormatPNG
	ImageFormatJPEG
)

func log(args ...interface{}) {
	fmt.Println(args...)
}

func resizeImage(data []byte, requestedWidth uint) (image.Image, ImageFormat, error) {
	log("Resizing image data (", len(data), "bytes)")

	start := time.Now()
	decodedImage, format, err := tryDecode(data)
	if err != nil {
		return nil, format, err
	}
	log("Decoding image data took", time.Now().Sub(start))

	start = time.Now()
	resizedImage := resize.Resize(requestedWidth, 0, decodedImage, resize.Lanczos3)

	log("Resizing image data took", time.Now().Sub(start))

	return resizedImage, format, err
}

func tryDecode(data []byte) (image.Image, ImageFormat, error) {
	img, err := png.Decode(bytes.NewBuffer(data))
	if err == nil {
		// image was a PNG
		return img, ImageFormatPNG, nil
	}

	img, err = jpeg.Decode(bytes.NewBuffer(data))
	if err == nil {
		// image was a JPEG
		return img, ImageFormatJPEG, nil
	}

	return nil, ImageFormatUnknown, err
}

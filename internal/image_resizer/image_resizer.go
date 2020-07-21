package image_resizer

import (
	"fmt"
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

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
	log("Resizing image data ...")

	decodedImage, format, err := tryDecode(data)
	if err != nil {
		return nil, format, err
	}

	resizedImage := resize.Resize(requestedWidth, 0, decodedImage, resize.Lanczos3)

	defer log("Finished loading image data, took", 10, "seconds")

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

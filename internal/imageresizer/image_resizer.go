package imageresizer

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"time"

	"github.com/nfnt/resize"
	"github.com/h2non/bimg"
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

func resizeImage(data []byte, requestedWidth uint, resizeImplementation string) (image.Image, ImageFormat, error) {
	log("Resizing image data (", len(data), "bytes)")

	start := time.Now()
	decodedImage, format, err := tryDecode(data)
	if err != nil {
		return nil, format, err
	}
	log("Decoding image data took", time.Now().Sub(start))

	start = time.Now()

	var resizedImage image.Image

	if resizeImplementation == "h2non/bimg" {
		log("Using `h2non/bimg` for resizing")

		log(requestedWidth)
		imgByte, err := bimg.NewImage(data).Resize(int(requestedWidth), 0)
		if err != nil {
			return nil, format, err
		}

		// TODO: We need an Image return type for our current API; It would be nice to profile and see if this cast is cheap or not
		resizedImage, _, err = image.Decode(bytes.NewReader(imgByte))
		if err != nil {
			return nil, format, err
		}
	} else {
		log("Using `nfnt/resize` for resizing")

		resizedImage = resize.Resize(requestedWidth, 0, decodedImage, resize.Lanczos3)
	}

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

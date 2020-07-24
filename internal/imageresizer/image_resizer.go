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

func logTiming(msg string, start time.Time) {
	log(msg, time.Now().Sub(start).Microseconds(), "mus")
}

func resizeImage(data []byte, requestedWidth uint, resizeImplementation string) ([]byte, ImageFormat, error) {
	log("Resizing image data (", len(data), "bytes)")

	if resizeImplementation == "nfnt/resize" {
		return nfntResize(data, requestedWidth)
	}
	
	return bimgResize(data, requestedWidth)
}

func bimgResize(data []byte, requestedWidth uint) ([]byte, ImageFormat, error) {
	log("Using `h2non/bimg` for resizing")

	var resizedImageData []byte
	var format ImageFormat
	var err error

	start := time.Now()
	var bimgfmt bimg.ImageType = bimg.DetermineImageType(data)
	switch bimgfmt {
	case bimg.JPEG:
		format = ImageFormatJPEG
	case bimg.PNG:
		format = ImageFormatPNG
	default:
		format = ImageFormatUnknown
	}

	options := bimg.Options{
		Width: int(requestedWidth),
		Interpolator: bimg.Bicubic,
	}
	resizedImageData, err = bimg.Resize(data, options)
	if err != nil {
		return nil, format, err
	}
	logTiming("Resizing image data took", start)

	return resizedImageData, format, err
}

func nfntResize(data []byte, requestedWidth uint) ([]byte, ImageFormat, error) {
	log("Using `nfnt/resize` for resizing")

	var resizedImageData []byte
	var format ImageFormat
	var err error

	start := time.Now()
	var decodedImage image.Image
	decodedImage, format, err = tryDecode(data)
	if err != nil {
		return nil, format, err
	}
	logTiming("Decoding image data took", start)

	start = time.Now()
	resizedImage := resize.Resize(requestedWidth, 0, decodedImage, resize.Lanczos3)
	logTiming("Resizing image data took", start)

	start = time.Now()
	var buffer bytes.Buffer
	switch format {
	case ImageFormatPNG:
		png.Encode(&buffer, resizedImage)
	case ImageFormatJPEG:
		jpeg.Encode(&buffer, resizedImage, nil)
	}
	logTiming("Re-encoding image data took", start)
	resizedImageData = buffer.Bytes()

	return resizedImageData, format, err
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

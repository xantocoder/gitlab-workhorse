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

	var bimgfmt bimg.ImageType = bimg.DetermineImageType(data)
	switch bimgfmt {
	case bimg.JPEG:
		format = ImageFormatJPEG
	case bimg.PNG:
		format = ImageFormatPNG
	default:
		format = ImageFormatUnknown
	}

	resizedImageData, err = bimg.NewImage(data).Resize(int(requestedWidth), 0)
	if err != nil {
		return nil, format, err
	}

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
	log("Decoding image data took", time.Now().Sub(start))

	start = time.Now()
	resizedImage := resize.Resize(requestedWidth, 0, decodedImage, resize.Lanczos3)
	log("Resizing image data took", time.Now().Sub(start))

	start = time.Now()
	buffer := new(bytes.Buffer)
	switch format {
	case ImageFormatPNG:
		png.Encode(buffer, resizedImage)
	case ImageFormatJPEG:
		jpeg.Encode(buffer, resizedImage, nil)
	}
	log("Re-encoding image data took", time.Now().Sub(start))
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

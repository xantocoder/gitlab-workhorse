// +build resizer_static_build

package main

/*
#cgo pkg-config: --static GraphicsMagickWand
#cgo pkg-config: --static GraphicsMagick
#cgo pkg-config: --static libjpeg
#cgo pkg-config: --static libwebpmux
#cgo pkg-config: --static libpng16
#cgo LDFLAGS: -static
*/
import "C"

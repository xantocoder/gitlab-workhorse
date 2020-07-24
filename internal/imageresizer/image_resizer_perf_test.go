package imageresizer

import (
	"io/ioutil"
	"testing"
	"os"
	"runtime"

	"github.com/stretchr/testify/require"
)

const (
	requestedWidthSmall = 16
	requestedWidthLarge = 200
)

func BenchmarkScaleSmallImage(b *testing.B) {
	benchmarkScaleImage("../../testdata/gitlab_small.png", b)
}

func BenchmarkScaleLargeImage(b *testing.B) {
	benchmarkScaleImage("../../testdata/gitlab_large.jpg", b)
}

func benchmarkScaleImage(filePath string, b *testing.B) {
	m := measureMemory(func() {
		file, err := os.Open(filePath)
		require.NoError(b, err)

		imageData, err := ioutil.ReadAll(file)
		require.NoError(b, err)

		_, _, err = resizeImage(imageData, requestedWidthSmall, "")
		require.NoError(b, err)
	})

	b.ReportMetric(m, "MiB/op")
}

func measureMemory(f func()) float64 {
	var m, m1 runtime.MemStats
	runtime.ReadMemStats(&m)

	f()

	runtime.ReadMemStats(&m1)

	return float64(m1.Alloc-m.Alloc) / 1024 / 1024
}
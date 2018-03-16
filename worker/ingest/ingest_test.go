package ingest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectPiazzaFileType(t *testing.T) {
	assert.Equal(t, "geojson", detectPiazzaFileType("something.geojson"))
	assert.Equal(t, "geojson", detectPiazzaFileType("SOMETHING_ELSE.GEOJSON"))
	assert.Equal(t, "raster", detectPiazzaFileType("image.tiff"))
	assert.Equal(t, "raster", detectPiazzaFileType("Image.GeoTiff"))
	assert.Equal(t, "text", detectPiazzaFileType("stuff.txt"))
	assert.Equal(t, "text", detectPiazzaFileType("abc123.unknownformat"))
	assert.Equal(t, "text", detectPiazzaFileType("no_extension"))
}

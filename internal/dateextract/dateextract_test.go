package dateextract_test

import (
	"testing"
	"time"

	"github.com/pinn/takesort/internal/dateextract"
	"github.com/stretchr/testify/assert"
)

func TestExtractDate_ValidDJIFilename(t *testing.T) {
	got, ok := dateextract.ExtractDate("DJI_20260218094732_0004_D.MP4")

	assert.True(t, ok)
	assert.Equal(t, time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC), got)
}

func TestExtractDate_CaseInsensitivePrefix(t *testing.T) {
	got, ok := dateextract.ExtractDate("dji_20260218094732_0004_D.mp4")

	assert.True(t, ok)
	assert.Equal(t, time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC), got)
}

func TestExtractDate_NonDJIFilenames(t *testing.T) {
	got1, ok1 := dateextract.ExtractDate("GoPro_0042.mp4")
	assert.False(t, ok1)
	assert.True(t, got1.IsZero())

	got2, ok2 := dateextract.ExtractDate("clip.mp4")
	assert.False(t, ok2)
	assert.True(t, got2.IsZero())
}

func TestExtractDate_InvalidDateDigits(t *testing.T) {
	got, ok := dateextract.ExtractDate("DJI_99991332000000_0001_D.MP4")

	assert.False(t, ok)
	assert.True(t, got.IsZero())
}

func TestExtractDate_ShortTimestamp(t *testing.T) {
	got, ok := dateextract.ExtractDate("DJI_2026021_0001_D.MP4")

	assert.False(t, ok)
	assert.True(t, got.IsZero())
}

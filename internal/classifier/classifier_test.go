package classifier_test

import (
	"testing"

	"github.com/pinn/takesort/internal/classifier"
	"github.com/stretchr/testify/assert"
)

func TestClassify_MP4IsVideo(t *testing.T) {
	assert.Equal(t, classifier.Video, classifier.Classify("clip.mp4"))
	assert.Equal(t, classifier.Video, classifier.Classify("CLIP.MP4"))
}

func TestClassify_ProxyCaseInsensitive(t *testing.T) {
	assert.Equal(t, classifier.Proxy, classifier.Classify("file.lrf"))
	assert.Equal(t, classifier.Proxy, classifier.Classify("file.LRF"))
	assert.Equal(t, classifier.Proxy, classifier.Classify("file.lrv"))
	assert.Equal(t, classifier.Proxy, classifier.Classify("file.LRV"))
}

func TestClassify_TrashExtensions(t *testing.T) {
	assert.Equal(t, classifier.Trash, classifier.Classify("file.thm"))
	assert.Equal(t, classifier.Trash, classifier.Classify("file.THM"))
	assert.Equal(t, classifier.Trash, classifier.Classify("file.srt"))
	assert.Equal(t, classifier.Trash, classifier.Classify("file.SRT"))
}

func TestClassify_PhotoAndAudio(t *testing.T) {
	assert.Equal(t, classifier.PhotoJPEG, classifier.Classify("photo.jpg"))
	assert.Equal(t, classifier.PhotoJPEG, classifier.Classify("photo.JPG"))
	assert.Equal(t, classifier.PhotoRAW, classifier.Classify("photo.dng"))
	assert.Equal(t, classifier.PhotoRAW, classifier.Classify("photo.DNG"))
	assert.Equal(t, classifier.Audio, classifier.Classify("sound.wav"))
	assert.Equal(t, classifier.Audio, classifier.Classify("sound.WAV"))
}

func TestClassify_UnknownExtension(t *testing.T) {
	assert.Equal(t, classifier.Unknown, classifier.Classify("file.xyz"))
	assert.Equal(t, classifier.Unknown, classifier.Classify("noext"))
	assert.Equal(t, classifier.Unknown, classifier.Classify(""))
}

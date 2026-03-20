package classifier

import (
	"path/filepath"
	"strings"
)

// FileType represents the classification of a media file.
type FileType int

const (
	Unknown   FileType = iota
	Video              // .mp4
	Proxy              // .lrf, .lrv
	PhotoJPEG          // .jpg
	PhotoRAW           // .dng
	Audio              // .wav
	Trash              // .thm, .srt
)

var extMap = map[string]FileType{
	".mp4": Video,
	".lrf": Proxy,
	".lrv": Proxy,
	".jpg": PhotoJPEG,
	".dng": PhotoRAW,
	".wav": Audio,
	".thm": Trash,
	".srt": Trash,
}

// Classify returns the FileType for a given filename based on its extension.
func Classify(filename string) FileType {
	ext := strings.ToLower(filepath.Ext(filename))
	if ft, ok := extMap[ext]; ok {
		return ft
	}
	return Unknown
}

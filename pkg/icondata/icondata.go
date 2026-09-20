// Package icondata converts icon bytes or files into data URLs. It knows
// nothing about processes, so other projects can reuse it as-is.
package icondata

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MaxBytes caps how many bytes a single icon file may contribute.
const MaxBytes = 256 << 10

// FileURL reads an icon file and returns its data URL, or "" when the file is
// missing, oversized, or not a decodable PNG or SVG image.
func FileURL(path string) string {
	data, err := ReadFile(path)
	if err != nil {
		return ""
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		url, ok := SVGURL(data)
		if !ok {
			return ""
		}
		return url
	}
	return PNGURL(data)
}

// SVGURL encodes an SVG document; ok is false unless the first element is an
// svg element in the SVG namespace.
func SVGURL(data []byte) (string, bool) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", false
		}
		if root, ok := token.(xml.StartElement); ok {
			if root.Name.Local != "svg" || root.Name.Space != "http://www.w3.org/2000/svg" {
				return "", false
			}
			return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data), true
		}
	}
}

// PNGURL encodes PNG data within MaxBytes and 1024x1024; anything else yields "".
func PNGURL(data []byte) string {
	if len(data) > MaxBytes {
		return ""
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width > 1024 || config.Height > 1024 {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

// ReadFile reads at most MaxBytes from path.
func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if len(data) > MaxBytes {
		return nil, io.ErrShortBuffer
	}
	return data, err
}

//go:generate go-enum --sql --marshal --file $GOFILE
package img

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/dsoprea/go-exif/v3"
	"github.com/marusama/semaphore/v2"

	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

// ErrUnsupportedFormat means the given image format is not supported.
var ErrUnsupportedFormat = errors.New("unsupported image format")

// ErrImageTooLarge means the image is too large to create a thumbnail.
var ErrImageTooLarge = errors.New("image too large for thumbnail generation")

// Maximum dimensions for thumbnail generation to prevent server crashes
const (
	MaxImageWidth  = 10000
	MaxImageHeight = 10000
)

// Service
type Service struct {
	sem semaphore.Semaphore
}

func New(workers int) *Service {
	return &Service{
		sem: semaphore.New(workers),
	}
}

// Format is an image file format.
/*
ENUM(
jpeg
png
gif
tiff
bmp
raw
)
*/
type Format int

func (x Format) toImaging() imaging.Format {
	switch x {
	case FormatJpeg:
		return imaging.JPEG
	case FormatPng:
		return imaging.PNG
	case FormatGif:
		return imaging.GIF
	case FormatTiff:
		return imaging.TIFF
	case FormatBmp:
		return imaging.BMP
	default:
		return imaging.JPEG
	}
}

/*
ENUM(
high
medium
low
)
*/
type Quality int

func (x Quality) resampleFilter() imaging.ResampleFilter {
	switch x {
	case QualityHigh:
		return imaging.Lanczos
	case QualityMedium:
		return imaging.Box
	case QualityLow:
		return imaging.NearestNeighbor
	default:
		return imaging.Box
	}
}

/*
ENUM(
fit
fill
)
*/
type ResizeMode int

func (s *Service) FormatFromExtension(ext string) (Format, error) {
	rawExts := map[string]bool{
		".arw": true, ".cr2": true, ".cr3": true, ".nef": true, ".nrw": true,
		".orf": true, ".raf": true, ".raw": true, ".rw2": true, ".dng": true,
	}
	if rawExts[strings.ToLower(ext)] {
		return FormatRaw, nil
	}

	format, err := imaging.FormatFromExtension(ext)
	if err != nil {
		return -1, ErrUnsupportedFormat
	}
	switch format {
	case imaging.JPEG:
		return FormatJpeg, nil
	case imaging.PNG:
		return FormatPng, nil
	case imaging.GIF:
		return FormatGif, nil
	case imaging.TIFF:
		return FormatTiff, nil
	case imaging.BMP:
		return FormatBmp, nil
	}

	return -1, ErrUnsupportedFormat
}

type resizeConfig struct {
	format     Format
	resizeMode ResizeMode
	quality    Quality
}

type Option func(*resizeConfig)

func WithFormat(format Format) Option {
	return func(config *resizeConfig) {
		config.format = format
	}
}

func WithMode(mode ResizeMode) Option {
	return func(config *resizeConfig) {
		config.resizeMode = mode
	}
}

func WithQuality(quality Quality) Option {
	return func(config *resizeConfig) {
		config.quality = quality
	}
}

func (s *Service) Resize(ctx context.Context, in io.Reader, width, height int, out io.Writer, options ...Option) error {
	if err := s.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.sem.Release(1)

	format, wrappedReader, err := s.detectFormat(in)
	if err != nil {
		return err
	}

	if format == FormatRaw {
		// For RAW files, extract the embedded JPEG preview
		preview, _, errRaw := s.extractRawPreview(wrappedReader)
		if errRaw == nil {
			// Do not attempt to use Go's image decoder to resize the extracted JPEG.
			// Camera manufacturer embedded JPEGs often contain makers notes or non-standard
			// markers that cause 'invalid JPEG format: unknown marker' in Go.
			// Browsers have tolerant native decoders, so we serve the embedded preview as-is.
			_, err := out.Write(preview)
			return err
		}
		return fmt.Errorf("failed to extract RAW preview: %w", errRaw)
	}

	config := resizeConfig{
		format:     format,
		resizeMode: ResizeModeFit,
		quality:    QualityMedium,
	}
	for _, option := range options {
		option(&config)
	}

	if config.quality == QualityLow && format == FormatJpeg {
		thm, newWrappedReader, errThm := getEmbeddedThumbnail(wrappedReader)
		wrappedReader = newWrappedReader
		if errThm == nil {
			_, err = out.Write(thm)
			if err == nil {
				return nil
			}
		}
	}

	img, err := imaging.Decode(wrappedReader, imaging.AutoOrientation(true))
	if err != nil {
		return err
	}

	switch config.resizeMode {
	case ResizeModeFill:
		img = imaging.Fill(img, width, height, imaging.Center, config.quality.resampleFilter())
	case ResizeModeFit:
		fallthrough
	default:
		img = imaging.Fit(img, width, height, config.quality.resampleFilter())
	}

	return imaging.Encode(out, img, config.format.toImaging())
}

func (s *Service) detectFormat(in io.Reader) (Format, io.Reader, error) {
	buf := &bytes.Buffer{}
	r := io.TeeReader(in, buf)

	imgConfig, imgFormat, err := image.DecodeConfig(r)
	if err != nil {
		// Check for RAW signatures if standard decode fails
		bufBytes := buf.Bytes()
		if isRaw(bufBytes) {
			return FormatRaw, io.MultiReader(buf, in), nil
		}
		return 0, nil, fmt.Errorf("%s: %w", err.Error(), ErrUnsupportedFormat)
	}

	// Check if image dimensions exceed maximum allowed size
	if imgConfig.Width > MaxImageWidth || imgConfig.Height > MaxImageHeight {
		return 0, nil, fmt.Errorf("image dimensions %dx%d exceed maximum %dx%d: %w",
			imgConfig.Width, imgConfig.Height, MaxImageWidth, MaxImageHeight, ErrImageTooLarge)
	}

	format, err := ParseFormat(imgFormat)
	if err != nil {
		return 0, nil, ErrUnsupportedFormat
	}

	return format, io.MultiReader(buf, in), nil
}

func getEmbeddedThumbnail(in io.Reader) ([]byte, io.Reader, error) {
	buf := &bytes.Buffer{}
	r := io.TeeReader(in, buf)
	wrappedReader := io.MultiReader(buf, in)

	offset := 0
	offsets := []int{12, 30}
	head := make([]byte, 0xffff)

	_, err := r.Read(head)
	if err != nil {
		return nil, wrappedReader, err
	}

	for _, offset = range offsets {
		if _, err = exif.ParseExifHeader(head[offset:]); err == nil {
			break
		}
	}

	if err != nil {
		return nil, wrappedReader, err
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return nil, wrappedReader, err
	}

	_, index, err := exif.Collect(im, exif.NewTagIndex(), head[offset:])
	if err != nil {
		return nil, wrappedReader, err
	}

	ifd := index.RootIfd.NextIfd()
	if ifd == nil {
		return nil, wrappedReader, exif.ErrNoThumbnail
	}

	thm, err := ifd.Thumbnail()
	return thm, wrappedReader, err
}

func (s *Service) extractRawPreview(in io.Reader) ([]byte, io.Reader, error) {
	buf := &bytes.Buffer{}
	r := io.TeeReader(in, buf)
	wrappedReader := io.MultiReader(buf, in)

	// Read enough to find EXIF/Previews (limit to 2MB for safety, though previews can be larger)
	// Some RAW files have previews at the end, but most are near the beginning.
	data := make([]byte, 2*1024*1024)
	n, _ := io.ReadFull(r, data)
	data = data[:n]

	// Use existing getEmbeddedThumbnail logic but potentially expanded
	// For now, reuse it as it already uses go-exif to find thumbnails/previews
	preview, _, err := getEmbeddedThumbnail(bytes.NewReader(data))
	if err == nil {
		return preview, wrappedReader, nil
	}

	// Fallback: search for JPEG markers in the first 2MB
	// JPEG starts with FF D8 and ends with FF D9
	start := bytes.Index(data, []byte{0xff, 0xd8})
	if start != -1 {
		end := bytes.LastIndex(data[start:], []byte{0xff, 0xd9})
		if end != -1 {
			return data[start : start+end+2], wrappedReader, nil
		}
	}

	return nil, wrappedReader, fmt.Errorf("no preview found")
}

func isRaw(data []byte) bool {
	if len(data) < 16 {
		return false
	}
	// TIFF containers (ARW, CR2, NEF, DNG)
	if (data[0] == 'I' && data[1] == 'I' && data[2] == '*') ||
		(data[0] == 'M' && data[1] == 'M' && data[2] == 0x00 && data[3] == '*') {
		return true
	}
	// CR3 (ISO BMFF)
	if bytes.Contains(data[:16], []byte("ftypcrx ")) {
		return true
	}
	// RAF
	if bytes.HasPrefix(data, []byte("FUJIFILMCCD-RAW ")) {
		return true
	}
	return false
}

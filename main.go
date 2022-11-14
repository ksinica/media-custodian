package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abema/go-mp4"
	"github.com/dsoprea/go-exif/v3"
	"lukechampine.com/blake3"
)

const (
	Blake3Size = 32
)

func findExifTag(entries []exif.ExifTag, name string) (interface{}, bool) {
	for _, x := range entries {
		if x.TagName == name {
			return x.Value, true
		}
	}
	return nil, false
}

func findExifDateTime(entries []exif.ExifTag) (string, bool) {
	if value, ok := findExifTag(entries, "DateTimeOriginal"); ok {
		return value.(string), true
	}
	if value, ok := findExifTag(entries, "DateTime"); ok {
		return value.(string), true
	}
	return "", false
}

func extensionFor(p string) string {
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".jpg", ".jpeg":
		return ".jpeg"
	default:
		return ext
	}
}

func guessNewImagePath(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := blake3.New(Blake3Size, nil)

	b, err := exif.SearchAndExtractExifWithReader(io.TeeReader(f, h))
	if err != nil {
		if errors.Is(err, exif.ErrNoExif) {
			return "", nil
		}
		return "", err
	}

	entries, _, err := exif.GetFlatExifDataUniversalSearch(b, nil, true)
	if err != nil {
		return "", err
	}

	dt, ok := findExifDateTime(entries)
	if !ok {
		return "", nil
	}

	t, err := time.Parse("2006:01:02 15:04:05", dt)
	if err != nil {
		return "", nil
	}

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Pictures/%s/%s-%s%s",
		t.Format("2006-01"),
		t.Format("20060102-150405"),
		hex.EncodeToString(h.Sum(nil)),
		extensionFor(p),
	), nil
}

func appleEpochToTime(epoch uint32) time.Time {
	return time.Unix(int64(epoch-2082844800), 0)
}

func findMP4DateTime(f *os.File) (ret time.Time, err error) {
	_, err = mp4.ReadBoxStructure(f, func(h *mp4.ReadHandle) (interface{}, error) {
		if h.BoxInfo.IsSupportedType() {
			box, _, err := h.ReadPayload()
			if err != nil {
				return nil, err
			}

			if box, ok := box.(*mp4.Mvhd); ok {
				ret = appleEpochToTime(box.CreationTimeV0)
				return nil, nil
			}

			_, err = h.Expand()
			if err == mp4.ErrUnsupportedBoxVersion {
				return nil, nil
			}
			return nil, err
		}
		return nil, nil
	})
	return
}

func guessNewVideoPath(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	t, err := findMP4DateTime(f)
	if err != nil {
		return "", err
	}

	if t.IsZero() {
		return "", nil
	}

	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}

	h := blake3.New(Blake3Size, nil)
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Videos/%s/%s-%s%s",
		t.Format("2006-01"),
		t.Format("20060102-150405"),
		hex.EncodeToString(h.Sum(nil)),
		extensionFor(p),
	), nil
}

func createDirectoryIfNotExists(p string) error {
	p = filepath.Dir(p)
	info, err := os.Stat(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err == nil && !info.IsDir() {
		return errors.New("not a directory")
	}
	return os.MkdirAll(p, 0755)
}

func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	n, err := f.Readdir(1)

	if len(n) == 0 && err == io.EOF {
		return true, nil
	}
	return false, err
}

func run(src, dest string) error {
	err := filepath.Walk(src, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			var np string
			switch strings.ToLower(filepath.Ext(info.Name())) {
			case ".jpg", ".jpeg", ".dng":
				np, err = guessNewImagePath(p)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: cannot guess new image path for %s: %s\n", p, err)
					return nil
				}

			case ".mp4":
				np, err = guessNewVideoPath(p)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: cannot guess new video path for %s: %s\n", p, err)
					return nil
				}

			default:
				return nil
			}

			if len(np) == 0 {
				fmt.Fprintln(os.Stderr, "Warning:", "no metadata for", p)
				return nil
			}

			np = filepath.Join(dest, np)

			if err := createDirectoryIfNotExists(np); err != nil {
				return err
			}

			_, err := os.Stat(np)
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(os.Stderr, "Warning:", p, "has a duplicate")
				return nil
			}

			if err := os.Rename(p, np); err != nil {
				return err
			}

			fmt.Fprintln(os.Stdout, "Moved", p, "to", np)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return filepath.Walk(src, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			ok, err := isDirEmpty(p)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Warning:", "could not determine if directory", p, "is empty:", err)
				return nil
			}

			if ok {
				if err := os.Remove(p); err != nil {
					fmt.Fprintln(os.Stderr, "Warning:", p, "directory cannot be removed:", err)
				} else {
					fmt.Fprintln(os.Stdout, p, "removed, because it's empty")
				}
			}
		}
		return nil
	})
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, " ", os.Args[0], "<source>", "<destination>")
		os.Exit(2)
	}

	src, dest := filepath.Clean(os.Args[1]), filepath.Clean(os.Args[2])

	if err := run(src, dest); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	os.Exit(0)
}

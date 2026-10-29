package walker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Options controls which files Walk yields.
type Options struct {
	// Exts matches file extensions (with or without leading dot), case-insensitive.
	Exts []string
	// NameGlob is a filepath.Match pattern against the base file name.
	NameGlob string
	// NameRegex is a regular expression matched against the base file name.
	NameRegex string
	// MinSize / MaxSize filter by file size in bytes. Zero means no bound.
	// MaxSize < 0 disables the upper bound check when MinSize is set alone;
	// use MaxSize == 0 for "no max". Non-zero MaxSize is inclusive.
	MinSize int64
	MaxSize int64
	// After / Before filter by modification time (inclusive). Zero means unset.
	After  time.Time
	Before time.Time
}

// FileInfo is a visited regular file.
type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
}

// Walk recursively scans root with filepath.WalkDir and invokes fn for each
// matching regular file. Walk continues after fn errors are not expected —
// fn should return an error only to abort the entire walk.
func Walk(root string, opts Options, fn func(FileInfo) error) error {
	re, err := compileRegex(opts.NameRegex)
	if err != nil {
		return err
	}
	exts := normalizeExts(opts.Exts)

	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("访问路径: %w", err)
	}
	if !info.IsDir() {
		fi := FileInfo{Path: root, Size: info.Size(), ModTime: info.ModTime()}
		if match(fi, filepath.Base(root), exts, opts, re) {
			return fn(fi)
		}
		return nil
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		fi := FileInfo{
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if !match(fi, d.Name(), exts, opts, re) {
			return nil
		}
		return fn(fi)
	})
}

// Collect walks root and returns all matching files.
func Collect(root string, opts Options) ([]FileInfo, error) {
	var out []FileInfo
	err := Walk(root, opts, func(fi FileInfo) error {
		out = append(out, fi)
		return nil
	})
	return out, err
}

func compileRegex(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("无效正则 --regex: %w", err)
	}
	return re, nil
}

func normalizeExts(exts []string) []string {
	if len(exts) == 0 {
		return nil
	}
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out = append(out, strings.ToLower(e))
	}
	return out
}

func match(fi FileInfo, base string, exts []string, opts Options, re *regexp.Regexp) bool {
	if len(exts) > 0 {
		ext := strings.ToLower(filepath.Ext(base))
		ok := false
		for _, e := range exts {
			if ext == e {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if opts.NameGlob != "" {
		ok, err := filepath.Match(opts.NameGlob, base)
		if err != nil || !ok {
			return false
		}
	}

	if re != nil && !re.MatchString(base) {
		return false
	}

	if opts.MinSize > 0 && fi.Size < opts.MinSize {
		return false
	}
	if opts.MaxSize > 0 && fi.Size > opts.MaxSize {
		return false
	}

	if !opts.After.IsZero() && fi.ModTime.Before(opts.After) {
		return false
	}
	if !opts.Before.IsZero() && fi.ModTime.After(opts.Before) {
		return false
	}

	return true
}

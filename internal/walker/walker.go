package walker

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Options controls which files Walk yields.
type Options struct {
	// Exclude matches basenames or slash-separated paths relative to root.
	// Matching directories are pruned. Excludes use path.Match on every OS (no **).
	Exclude []string
	// Exts matches file extensions (with or without leading dot), case-insensitive.
	Exts []string
	// NameGlob is a filepath.Match pattern against the base file name.
	NameGlob string
	// NameRegex is a regular expression matched against the base file name.
	NameRegex string
	// MinSize / MaxSize filter by file size in bytes. Zero means no bound.
	// Negative limits are invalid. Non-zero MaxSize is inclusive.
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
// matching regular file. Filesystem errors are collected while accessible siblings
// continue, then returned together. Callback errors abort immediately. Symlinks,
// including a symlink passed as root, are skipped.
func Walk(root string, opts Options, fn func(FileInfo) error) error {
	if _, err := filepath.Match(opts.NameGlob, ""); err != nil {
		return fmt.Errorf("invalid name glob %q: %w", opts.NameGlob, err)
	}
	for _, pattern := range opts.Exclude {
		if _, err := path.Match(pattern, ""); err != nil {
			return fmt.Errorf("invalid glob %q: %w", pattern, err)
		}
	}
	if opts.MinSize < 0 || opts.MaxSize < 0 || (opts.MaxSize > 0 && opts.MinSize > opts.MaxSize) {
		return fmt.Errorf("invalid size range")
	}
	if !opts.After.IsZero() && !opts.Before.IsZero() && opts.After.After(opts.Before) {
		return fmt.Errorf("--after must not be later than --before")
	}
	re, err := compileRegex(opts.NameRegex)
	if err != nil {
		return err
	}
	exts := normalizeExts(opts.Exts)

	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("访问路径: %w", err)
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() || excluded(filepath.Base(root), filepath.Base(root), opts.Exclude) {
			return nil
		}
		fi := FileInfo{Path: root, Size: info.Size(), ModTime: info.ModTime()}
		if match(fi, filepath.Base(root), exts, opts, re) {
			return fn(fi)
		}
		return nil
	}

	var scanErrors []error
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			scanErrors = append(scanErrors, walkErr)
			return nil
		}
		if path != filepath.Clean(root) {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if excluded(d.Name(), filepath.ToSlash(rel), opts.Exclude) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("%s: %w", path, err))
			return nil
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
	return errors.Join(append(scanErrors, err)...)
}

func excluded(base, relative string, patterns []string) bool {
	for _, pattern := range patterns {
		candidate := base
		if strings.Contains(pattern, "/") {
			candidate = relative
		}
		if ok, _ := path.Match(pattern, candidate); ok {
			return true
		}
	}
	return false
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

package renameutil

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Options controls how a basename is transformed.
type Options struct {
	Prefix   string
	Suffix   string // inserted before the extension
	Find     string // regex matched against the full basename
	Repl     string
	Seq      bool
	SeqStart int // inclusive; used as start + index
	SeqWidth int // zero-padding width; values < 1 become 1

	findRe *regexp.Regexp
}

// Compile validates and compiles Find. Call once before Plan/Apply.
func (o *Options) Compile() error {
	if o.Find == "" {
		return nil
	}
	re, err := regexp.Compile(o.Find)
	if err != nil {
		return fmt.Errorf("无效重命名正则 --find: %w", err)
	}
	o.findRe = re
	return nil
}

// HasRule reports whether any rename rule is configured.
func (o Options) HasRule() bool {
	return o.Prefix != "" || o.Suffix != "" || o.Find != "" || o.Seq
}

// Apply transforms a single basename. index is 0-based among the planned set.
func (o Options) Apply(basename string, index int) (string, error) {
	if basename == "" || basename == "." || basename == ".." {
		return "", fmt.Errorf("无效文件名 %q", basename)
	}

	name := basename
	if o.findRe != nil {
		name = o.findRe.ReplaceAllString(name, o.Repl)
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	base = o.Prefix + base + o.Suffix

	if o.Seq {
		width := o.SeqWidth
		if width < 1 {
			width = 1
		}
		start := o.SeqStart
		base = fmt.Sprintf("%s_%0*d", base, width, start+index)
	}

	out := base + ext
	if out == "" || out == "." || out == ".." {
		return "", fmt.Errorf("重命名结果无效: %q (来自 %s)", out, basename)
	}
	if strings.ContainsAny(out, `/\`) {
		return "", fmt.Errorf("重命名结果含非法路径字符: %q", out)
	}
	return out, nil
}

// Mapping is one planned rename.
type Mapping struct {
	OldPath string
	NewPath string
	OldName string
	NewName string
	Index   int
	Skipped bool // true when OldPath == NewPath
}

// Plan builds rename mappings for paths (absolute or relative file paths).
// paths should already be sorted if deterministic sequence numbers are desired.
func Plan(paths []string, opts Options) ([]Mapping, error) {
	if err := opts.Compile(); err != nil {
		return nil, err
	}
	if !opts.HasRule() {
		return nil, fmt.Errorf("请至少指定一种重命名规则：--prefix / --suffix / --find / --seq")
	}

	out := make([]Mapping, 0, len(paths))
	seen := make(map[string]string, len(paths)) // newPath -> oldPath

	for i, oldPath := range paths {
		dir := filepath.Dir(oldPath)
		oldName := filepath.Base(oldPath)
		newName, err := opts.Apply(oldName, i)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", oldPath, err)
		}
		newPath := filepath.Join(dir, newName)
		m := Mapping{
			OldPath: oldPath,
			NewPath: newPath,
			OldName: oldName,
			NewName: newName,
			Index:   i,
			Skipped: filepath.Clean(oldPath) == filepath.Clean(newPath),
		}
		if !m.Skipped {
			if prev, ok := seen[filepath.Clean(newPath)]; ok {
				return nil, fmt.Errorf("目标冲突: %s 与 %s 都映射到 %s", prev, oldPath, newPath)
			}
			seen[filepath.Clean(newPath)] = oldPath
		}
		out = append(out, m)
	}
	return out, nil
}

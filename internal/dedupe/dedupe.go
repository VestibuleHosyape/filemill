// Package dedupe plans duplicate removal without modifying files during scanning.
package dedupe

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"

	"filemill/internal/hashutil"
	"filemill/internal/walker"
	"filemill/internal/worker"
)

type File struct {
	Path string `json:"path"`
	info os.FileInfo
}

type Group struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Keep       File   `json:"keep"`
	Duplicates []File `json:"duplicates"`
}

type Failure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type Result struct {
	Groups           []Group   `json:"groups"`
	Errors           []Failure `json:"errors"`
	Scanned          int       `json:"scanned"`
	Hashed           int       `json:"hashed"`
	ReclaimableBytes int64     `json:"reclaimable_bytes"`
}

// Find hashes only non-unique sizes. Results are independent of worker order.
// ReclaimableBytes counts logical bytes, not physical allocation or hardlink savings.
func Find(files []walker.FileInfo, workers int) Result {
	r := Result{Groups: []Group{}, Errors: []Failure{}, Scanned: len(files)}
	bySize := make(map[int64][]walker.FileInfo)
	for _, f := range files {
		bySize[f.Size] = append(bySize[f.Size], f)
	}
	type hashed struct {
		file File
		sum  string
		size int64
	}
	var found []hashed
	var mu sync.Mutex
	pool := worker.New(workers)
	pool.Start()
	for _, candidates := range bySize {
		if len(candidates) < 2 {
			continue
		}
		for _, candidate := range candidates {
			r.Hashed++
			_ = pool.Submit(func() {
				f, sum, err := fingerprint(candidate.Path)
				mu.Lock()
				defer mu.Unlock()
				if err == nil && (f.info.Size() != candidate.Size || !f.info.ModTime().Equal(candidate.ModTime)) {
					err = fmt.Errorf("file changed since scan")
				}
				if err != nil {
					r.Errors = append(r.Errors, Failure{candidate.Path, err.Error()})
					return
				}
				found = append(found, hashed{f, sum, candidate.Size})
			})
		}
	}
	pool.Close()
	sort.Slice(found, func(i, j int) bool { return found[i].file.Path < found[j].file.Path })
	groups := make(map[string]*Group)
	for _, h := range found {
		key := fmt.Sprintf("%d:%s", h.size, h.sum)
		g := groups[key]
		if g == nil {
			groups[key] = &Group{Hash: h.sum, Size: h.size, Keep: h.file, Duplicates: []File{}}
			continue
		}
		// An alias of the keeper is not a redundant independent copy.
		if os.SameFile(g.Keep.info, h.file.info) {
			continue
		}
		g.Duplicates = append(g.Duplicates, h.file)
	}
	for _, g := range groups {
		if len(g.Duplicates) == 0 {
			continue
		}
		r.Groups = append(r.Groups, *g)
		r.ReclaimableBytes += g.Size * int64(len(g.Duplicates))
	}
	sort.Slice(r.Groups, func(i, j int) bool { return r.Groups[i].Keep.Path < r.Groups[j].Keep.Path })
	sort.Slice(r.Errors, func(i, j int) bool { return r.Errors[i].Path < r.Errors[j].Path })
	return r
}

func unchanged(before, after os.FileInfo) bool {
	return before != nil && after != nil && after.Mode().IsRegular() &&
		os.SameFile(before, after) && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

func openRegular(path string) (*os.File, os.FileInfo, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("not a regular file: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	after, err := f.Stat()
	if err != nil || !unchanged(before, after) {
		f.Close()
		return nil, nil, fmt.Errorf("file changed while opening: %s", path)
	}
	return f, after, nil
}

func fingerprint(path string) (File, string, error) {
	f, before, err := openRegular(path)
	if err != nil {
		return File{}, "", err
	}
	defer f.Close()
	h, _ := hashutil.New(hashutil.SHA256)
	if _, err := io.Copy(h, f); err != nil {
		return File{}, "", err
	}
	after, err := f.Stat()
	if err != nil || !unchanged(before, after) {
		return File{}, "", fmt.Errorf("file changed while hashing: %s", path)
	}
	current, err := os.Lstat(path)
	if err != nil || !unchanged(before, current) {
		return File{}, "", fmt.Errorf("path changed while hashing: %s", path)
	}
	return File{Path: path, info: before}, hex.EncodeToString(h.Sum(nil)), nil
}

// RemoveVerified revalidates both snapshots, SHA-256 and byte equality before
// removing a duplicate. Use only on a quiescent tree: portable pathname deletion
// cannot eliminate a concurrent replacement between the last check and Remove.
func RemoveVerified(g Group, duplicate File) error {
	member := false
	for _, f := range g.Duplicates {
		if f.Path == duplicate.Path {
			member = true
			break
		}
	}
	if !member {
		return fmt.Errorf("file is not in the duplicate plan")
	}
	for _, expected := range []File{g.Keep, duplicate} {
		actual, sum, err := fingerprint(expected.Path)
		if err != nil {
			return err
		}
		if !unchanged(expected.info, actual.info) || sum != g.Hash {
			return fmt.Errorf("file changed since plan: %s", expected.Path)
		}
	}
	if os.SameFile(g.Keep.info, duplicate.info) {
		return fmt.Errorf("refusing to remove keeper alias")
	}
	a, ai, err := openRegular(g.Keep.Path)
	if err != nil {
		return err
	}
	defer a.Close()
	b, bi, err := openRegular(duplicate.Path)
	if err != nil {
		return err
	}
	defer b.Close()
	if !unchanged(g.Keep.info, ai) || !unchanged(duplicate.info, bi) {
		return fmt.Errorf("file changed before comparison")
	}
	ab, bb := make([]byte, 64*1024), make([]byte, 64*1024)
	for {
		an, ae := io.ReadFull(a, ab)
		bn, be := io.ReadFull(b, bb)
		if an != bn || !bytes.Equal(ab[:an], bb[:bn]) {
			return fmt.Errorf("contents differ")
		}
		if ae != nil && ae != io.EOF && ae != io.ErrUnexpectedEOF {
			return ae
		}
		if be != nil && be != io.EOF && be != io.ErrUnexpectedEOF {
			return be
		}
		if ae != nil || be != nil {
			if ae != be {
				return fmt.Errorf("contents differ")
			}
			break
		}
	}
	// Close before deleting for Windows compatibility.
	if err := b.Close(); err != nil {
		return err
	}
	for _, expected := range []File{g.Keep, duplicate} {
		current, err := os.Lstat(expected.Path)
		if err != nil || !unchanged(expected.info, current) {
			return fmt.Errorf("file changed before deletion: %s", expected.Path)
		}
	}
	return os.Remove(duplicate.Path)
}

// Command release builds archives locally; it never publishes or creates tags.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	version := flag.String("version", "v0.1.0-dev", "version embedded into binaries")
	out := flag.String("out", "dist", "archive directory")
	flag.Parse()
	if err := build(*version, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(version, out string) error {
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`).MatchString(version) {
		return fmt.Errorf("invalid version %q", version)
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(out, ".build-")
	if err != nil {
		return err
	}
	// stage is a newly created, uniquely named child of out owned by this run.
	defer os.RemoveAll(stage)
	var checksums strings.Builder
	for _, target := range []struct{ os, arch string }{{"windows", "amd64"}, {"windows", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}, {"darwin", "amd64"}, {"darwin", "arm64"}} {
		binary := "filemill"
		if target.os == "windows" {
			binary += ".exe"
		}
		binaryPath := filepath.Join(stage, binary)
		c := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w -X filemill/cmd.Version="+version, "-o", binaryPath, ".")
		for _, entry := range os.Environ() {
			key, _, _ := strings.Cut(entry, "=")
			if !strings.EqualFold(key, "GOOS") && !strings.EqualFold(key, "GOARCH") && !strings.EqualFold(key, "CGO_ENABLED") {
				c.Env = append(c.Env, entry)
			}
		}
		c.Env = append(c.Env, "GOOS="+target.os, "GOARCH="+target.arch, "CGO_ENABLED=0")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("build %s/%s: %w", target.os, target.arch, err)
		}
		name := fmt.Sprintf("filemill_%s_%s_%s", version, target.os, target.arch)
		isZip := target.os == "windows"
		if isZip {
			name += ".zip"
		} else {
			name += ".tar.gz"
		}
		path := filepath.Join(out, name)
		if err := archive(path, isZip, []string{binaryPath, "README.md", "README-CN.md", "LICENSE"}); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintf(&checksums, "%x  %s\n", h.Sum(nil), name)
		fmt.Println(path)
	}
	return os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(checksums.String()), 0644)
}

func archive(path string, isZip bool, inputs []string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	var zw *zip.Writer
	var tw *tar.Writer
	if isZip {
		zw = zip.NewWriter(f)
		defer func() {
			if e := zw.Close(); err == nil {
				err = e
			}
		}()
	} else {
		gz := gzip.NewWriter(f)
		defer func() {
			if e := gz.Close(); err == nil {
				err = e
			}
		}()
		tw = tar.NewWriter(gz)
		defer func() {
			if e := tw.Close(); err == nil {
				err = e
			}
		}()
	}
	for i, path := range inputs {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := int64(0644)
		if i == 0 {
			mode = 0755
		}
		name := filepath.Base(path)
		var w io.Writer
		if zw != nil {
			h := &zip.FileHeader{Name: name, Method: zip.Deflate}
			h.SetMode(os.FileMode(mode))
			w, err = zw.CreateHeader(h)
			if err != nil {
				return err
			}
		} else {
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data))}); err != nil {
				return err
			}
			w = tw
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

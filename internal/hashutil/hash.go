package hashutil

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// Supported algorithms.
const (
	MD5    = "md5"
	SHA1   = "sha1"
	SHA256 = "sha256"
)

// New returns a hash.Hash for a known algorithm name (case-insensitive).
func New(algo string) (hash.Hash, error) {
	switch strings.ToLower(strings.TrimSpace(algo)) {
	case MD5:
		return md5.New(), nil
	case SHA1, "sha-1":
		return sha1.New(), nil
	case SHA256, "sha-256":
		return sha256.New(), nil
	default:
		return nil, fmt.Errorf("不支持的哈希算法 %q（支持 md5 / sha1 / sha256）", algo)
	}
}

// Normalize returns the canonical algorithm name.
func Normalize(algo string) (string, error) {
	h, err := New(algo)
	if err != nil {
		return "", err
	}
	switch h.Size() {
	case md5.Size:
		return MD5, nil
	case sha1.Size:
		return SHA1, nil
	case sha256.Size:
		return SHA256, nil
	default:
		return strings.ToLower(algo), nil
	}
}

// File computes the hex digest of path using algo.
func File(path, algo string) (string, error) {
	h, err := New(algo)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开文件: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("读取文件: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

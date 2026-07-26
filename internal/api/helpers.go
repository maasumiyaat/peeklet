package api

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// slug characters: lowercase + digits, minus ambiguous ones.
const slugAlphabet = "abcdefghijkmnpqrstuvwxyz23456789"

func generateSlug(n int) (string, error) {
	out := make([]byte, n)
	max := big.NewInt(int64(len(slugAlphabet)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = slugAlphabet[idx.Int64()]
	}
	return string(out), nil
}

// normalizePrefix produces a safe "foo/bar/" prefix, or "" if invalid.
func normalizePrefix(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "/")
	if p == "" || strings.Contains(p, "..") {
		return ""
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

// mediaType classifies a filename as "video" or "image" by extension.
func mediaType(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return "image"
	}
	switch strings.ToLower(name[i+1:]) {
	case "mp4", "webm", "mov", "m4v", "ogg":
		return "video"
	default:
		return "image"
	}
}

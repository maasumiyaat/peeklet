package api

import "testing"

func TestNormalizePrefix(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare folder gets a trailing slash", "foo/bar", "foo/bar/"},
		{"leading slash is trimmed", "/foo/bar", "foo/bar/"},
		{"already trailing-slashed is left alone", "foo/bar/", "foo/bar/"},
		{"surrounding whitespace is trimmed", "  foo/bar  ", "foo/bar/"},
		{"empty string is invalid", "", ""},
		{"whitespace only is invalid", "   ", ""},
		{"dotdot traversal is rejected", "../etc", ""},
		{"embedded dotdot is rejected", "foo/../bar", ""},
		{"single top-level segment is valid", "foo", "foo/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizePrefix(tc.in)
			if got != tc.want {
				t.Fatalf("normalizePrefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMediaType(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"mp4 is video", "clip.mp4", "video"},
		{"webm is video", "clip.webm", "video"},
		{"mov is video", "clip.mov", "video"},
		{"m4v is video", "clip.m4v", "video"},
		{"ogg is video", "clip.ogg", "video"},
		{"uppercase video extension is still video", "CLIP.MP4", "video"},
		{"jpg is image", "photo.jpg", "image"},
		{"png is image", "photo.png", "image"},
		{"unknown extension defaults to image", "file.xyz", "image"},
		{"no extension defaults to image", "noext", "image"},
		{"dotfile with no suffix defaults to image", "trailing.", "image"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mediaType(tc.in)
			if got != tc.want {
				t.Fatalf("mediaType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

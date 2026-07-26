package otp

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

	cases := []struct {
		name   string
		length int
	}{
		{"typical length", 8},
		{"minimum sane length", 4},
		{"longer length", 32},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := Generate(tc.length, alphabet)
			if err != nil {
				t.Fatalf("Generate(%d, alphabet) error = %v", tc.length, err)
			}
			if len(code) != tc.length {
				t.Fatalf("Generate(%d, alphabet) returned length %d, want %d", tc.length, len(code), tc.length)
			}
			for _, r := range code {
				if !strings.ContainsRune(alphabet, r) {
					t.Fatalf("Generate returned %q, contains rune %q outside alphabet %q", code, r, alphabet)
				}
			}
		})
	}
}

func TestGenerateUsesFullAlphabetAcrossManyDraws(t *testing.T) {
	const alphabet = "AB"
	seen := map[rune]bool{}
	for i := 0; i < 200; i++ {
		code, err := Generate(1, alphabet)
		if err != nil {
			t.Fatalf("Generate error = %v", err)
		}
		seen[rune(code[0])] = true
	}
	if len(seen) != len(alphabet) {
		t.Fatalf("expected to observe all %d alphabet characters across 200 draws, saw %d", len(alphabet), len(seen))
	}
}

func TestHashVerifyRoundTrip(t *testing.T) {
	const code = "ABCD1234"

	hash, salt, err := Hash(code)
	if err != nil {
		t.Fatalf("Hash(%q) error = %v", code, err)
	}
	if hash == "" || salt == "" {
		t.Fatalf("Hash(%q) returned empty hash or salt", code)
	}
	if !Verify(code, hash, salt) {
		t.Fatalf("Verify(%q, hash, salt) = false, want true for the code that produced the hash", code)
	}
}

func TestVerifyRejectsWrongCode(t *testing.T) {
	hash, salt, err := Hash("ABCD1234")
	if err != nil {
		t.Fatalf("Hash error = %v", err)
	}

	cases := []struct {
		name      string
		candidate string
	}{
		{"different code", "WXYZ9999"},
		{"empty code", ""},
		{"prefix of the real code", "ABCD"},
		{"correct code with different case", "abcd1234"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if Verify(tc.candidate, hash, salt) {
				t.Fatalf("Verify(%q, hash, salt) = true, want false", tc.candidate)
			}
		})
	}
}

func TestVerifyRejectsTamperedHashOrSalt(t *testing.T) {
	const code = "ABCD1234"
	hash, salt, err := Hash(code)
	if err != nil {
		t.Fatalf("Hash error = %v", err)
	}

	if Verify(code, "not-hex", salt) {
		t.Fatalf("Verify with malformed hash should fail")
	}
	if Verify(code, hash, "not-hex") {
		t.Fatalf("Verify with malformed salt should fail")
	}
}

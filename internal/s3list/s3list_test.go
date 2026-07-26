package s3list

import (
	"errors"
	"testing"
)

func TestResolve(t *testing.T) {
	const granted = "clients/wedding-jan/"

	cases := []struct {
		name      string
		requested string
		want      string
		wantErr   error
	}{
		{"empty request resolves to root", "", granted, nil},
		{"granted root itself", "clients/wedding-jan/", granted, nil},
		{"subfolder within grant", "clients/wedding-jan/sub", "clients/wedding-jan/sub/", nil},
		{"nested subfolder within grant", "clients/wedding-jan/sub/sub2", "clients/wedding-jan/sub/sub2/", nil},
		{"dotdot climbing above root is harmless and lands outside grant", "../other", "", ErrOutsidePrefix},
		{"dotdot climbing out of the grant itself", "clients/wedding-jan/../../etc", "", ErrOutsidePrefix},
		{"dotdot within the grant that stays inside", "clients/wedding-jan/sub/../sub2", "clients/wedding-jan/sub2/", nil},
		{"sibling prefix that merely shares a string prefix is rejected", "clients/wedding-jan-evil/", "", ErrOutsidePrefix},
		{"unrelated path is rejected", "someone-else/folder/", "", ErrOutsidePrefix},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(granted, tc.requested)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Resolve(%q, %q) error = %v, want %v", granted, tc.requested, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("Resolve(%q, %q) = %q, want %q", granted, tc.requested, got, tc.want)
			}
		})
	}
}

func TestParent(t *testing.T) {
	const granted = "clients/wedding-jan/"

	cases := []struct {
		name    string
		current string
		want    string
	}{
		{"at the granted ceiling has no parent", granted, ""},
		{"one level below the ceiling returns the ceiling", "clients/wedding-jan/sub/", granted},
		{"two levels below returns the immediate parent", "clients/wedding-jan/sub/sub2/", "clients/wedding-jan/sub/"},
		{"current outside the grant has no parent", "someone-else/folder/", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Parent(granted, tc.current)
			if got != tc.want {
				t.Fatalf("Parent(%q, %q) = %q, want %q", granted, tc.current, got, tc.want)
			}
		})
	}
}

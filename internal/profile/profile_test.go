package profile

import (
	"errors"
	"testing"
)

func TestValidateName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"dot", ".", true},
		{"dotdot", "..", true},
		{"slash", "foo/bar", true},
		{"backslash", `foo\bar`, true},
		{"path_escape", "../evil", true},
		{"simple", "work", false},
		{"with_hyphen", "work-acme", false},
		{"with_under", "work_acme", false},
		{"alphanumeric", "client1", false},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateName(%q): want error, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateName(%q): want nil, got %v", tt.input, err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrInvalidName) {
				t.Fatalf("ValidateName(%q): want ErrInvalidName, got %v", tt.input, err)
			}
		})
	}
}

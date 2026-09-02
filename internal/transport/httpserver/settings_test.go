package httpserver

import "testing"

func TestNormalizeMemoryLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "gigabytes", input: " 4G ", want: "4g"},
		{name: "megabytes", input: "3072m", want: "3072m"},
		{name: "clear", input: "", want: ""},
		{name: "zero", input: "0g", wantErr: true},
		{name: "missing unit", input: "4", wantErr: true},
		{name: "shell characters", input: "4g; echo unsafe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeMemoryLimit(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeMemoryLimit(%q) expected an error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeMemoryLimit(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeMemoryLimit(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

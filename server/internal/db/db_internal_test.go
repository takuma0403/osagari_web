package db

import "testing"

func TestIsRemote(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://db:8080", true},
		{"https://example.turso.io", true},
		{"libsql://example.turso.io?authToken=x", true},
		{"file:/tmp/test.db", false},
	}
	for _, tt := range tests {
		if got := isRemote(tt.url); got != tt.want {
			t.Errorf("isRemote(%q) = %t, want %t", tt.url, got, tt.want)
		}
	}
}

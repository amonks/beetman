package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSkippedAlbums(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "beet-import-manager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock flac directory
	albumsDir := filepath.Join(tmpDir, "files/flac")
	if err := os.MkdirAll(albumsDir, 0755); err != nil {
		t.Fatalf("Failed to create flac dir: %v", err)
	}

	tests := []struct {
		name       string
		logContent []string
		want       map[string]string
	}{
		{
			name: "basic skip with absolute paths",
			logContent: []string{
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "album1")),
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "album2")),
			},
			want: map[string]string{
				"album1": "no strong match",
				"album2": "no strong match",
			},
		},
		{
			name: "duplicate-skip entries",
			logContent: []string{
				fmt.Sprintf("duplicate-skip %s", filepath.Join(albumsDir, "album1")),
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "album2")),
			},
			want: map[string]string{
				"album1": "duplicate",
				"album2": "no strong match",
			},
		},
		{
			name: "no skips",
			logContent: []string{
				fmt.Sprintf("added %s", filepath.Join(albumsDir, "album1")),
				fmt.Sprintf("added %s", filepath.Join(albumsDir, "album2")),
			},
			want: nil,
		},
		{
			name: "mixed content",
			logContent: []string{
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "album1")),
				fmt.Sprintf("added %s", filepath.Join(albumsDir, "album2")),
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "album3")),
			},
			want: map[string]string{
				"album1": "no strong match",
				"album3": "no strong match",
			},
		},
		{
			name: "skip with spaces",
			logContent: []string{
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "album with spaces")),
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "another album")),
			},
			want: map[string]string{
				"album with spaces": "no strong match",
				"another album":     "no strong match",
			},
		},
		{
			name:       "empty log file",
			logContent: []string{},
			want:       nil,
		},
		{
			// Multi-disc albums: beets logs each disc subdirectory as the
			// skipped path (joined by "; "). The key must be the album
			// directory, not the disc subdirectory, or the manager can't
			// match it against the batch and mismarks the album imported.
			name: "multi-disc skip paths map to the album directory",
			logContent: []string{
				fmt.Sprintf("skip %s; %s",
					filepath.Join(albumsDir, "boxset_album", "Disc 1"),
					filepath.Join(albumsDir, "boxset_album", "Disc 2")),
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "single_disc_album", "CD1")),
				fmt.Sprintf("duplicate-skip %s", filepath.Join(albumsDir, "dup_box", "2 - Studio CD2")),
			},
			want: map[string]string{
				"boxset_album":      "no strong match",
				"single_disc_album": "no strong match",
				"dup_box":           "duplicate",
			},
		},
		{
			// Beets joins multiple paths with "; ", but "; " can also appear
			// inside a directory name ("Joana Gama; Luís Fernandes - ...").
			// Only "; " followed by the start of another absolute path is a
			// separator.
			name: "semicolon in album directory name",
			logContent: []string{
				fmt.Sprintf("skip %s", filepath.Join(albumsDir, "Joana Gama; Luís Fernandes - Quest")),
				fmt.Sprintf("skip %s; %s",
					filepath.Join(albumsDir, "Artist A; Artist B - Split", "Disc 1"),
					filepath.Join(albumsDir, "Artist A; Artist B - Split", "Disc 2")),
			},
			want: map[string]string{
				"Joana Gama; Luís Fernandes - Quest": "no strong match",
				"Artist A; Artist B - Split":         "no strong match",
			},
		},
	}

	parser := New(albumsDir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test log file
			logFile := filepath.Join(tmpDir, "test.log")
			if err := os.WriteFile(logFile, []byte(strings.Join(tt.logContent, "\n")), 0644); err != nil {
				t.Fatalf("Failed to write test log file: %v", err)
			}

			got, err := parser.ParseSkippedAlbums(logFile)
			if err != nil {
				t.Fatalf("ParseSkippedAlbums() error = %v", err)
			}

			// Compare results
			if len(got) != len(tt.want) {
				t.Errorf("ParseSkippedAlbums() got %v items, want %v items\ngot: %v\nwant: %v", len(got), len(tt.want), got, tt.want)
				return
			}

			for album, wantReason := range tt.want {
				gotReason, ok := got[album]
				if !ok {
					t.Errorf("ParseSkippedAlbums() missing expected album %q", album)
				} else if gotReason != wantReason {
					t.Errorf("ParseSkippedAlbums() album %q reason = %q, want %q", album, gotReason, wantReason)
				}
			}
		})
	}

	// Test non-existent log file
	t.Run("non-existent log file", func(t *testing.T) {
		got, err := parser.ParseSkippedAlbums(filepath.Join(tmpDir, "nonexistent.log"))
		if err != nil {
			t.Errorf("ParseSkippedAlbums() error = %v", err)
		}
		if got != nil {
			t.Errorf("ParseSkippedAlbums() = %v, want nil", got)
		}
	})
}

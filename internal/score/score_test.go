package score

import "testing"

func TestMatchBasic(t *testing.T) {
	tests := []struct {
		text, pat string
		want      int // -1 = no match; otherwise just > 0
	}{
		{"readme", "readme", 1},
		{"readme.md", "readme", 1},
		{"README.md", "readme", -1}, // case-sensitive at this layer
		{"my_project.go", "proj", 1},
		{"unrelated", "xyz", -1},
	}
	for _, tt := range tests {
		got := Match(tt.text, tt.pat)
		if tt.want < 0 && got != -1 {
			t.Errorf("Match(%q,%q) = %d, want -1", tt.text, tt.pat, got)
		}
		if tt.want > 0 && got <= 0 {
			t.Errorf("Match(%q,%q) = %d, want >0", tt.text, tt.pat, got)
		}
	}
}

func TestMatchPrefersBoundary(t *testing.T) {
	a := Match("my_project.go", "proj")
	b := Match("subproject_xyz", "proj")
	if a <= b {
		t.Errorf("boundary match should win: a=%d b=%d", a, b)
	}
}

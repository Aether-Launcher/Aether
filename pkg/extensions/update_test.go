package extensions

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.99.99", 1},
		{"v1.0.0-beta.15", "v1.0.0-beta.14", 1},
		{"v1.0.0-beta.14", "v1.0.0-beta.15", -1},
		{"v1.0.0-beta.15", "v1.0.0-beta.15", 0},
		{"1.0.0", "1.0.0-beta.15", 1},
		{"1.0.0-beta.15", "1.0.0", -1},
		{"v1.0.0-alpha.1", "v1.0.0-beta.1", -1},
	}

	for _, tc := range tests {
		got := CompareVersions(tc.a, tc.b)
		if got != tc.expected {
			t.Errorf("CompareVersions(%q, %q) = %d; want %d", tc.a, tc.b, got, tc.expected)
		}
	}
}

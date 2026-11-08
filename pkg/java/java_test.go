package java

import "testing"

func TestRequiredJavaVersion(t *testing.T) {
	tests := []struct {
		mc   string
		want int
	}{
		{"", 8},
		{"1.16.5", 8},
		{"1.17", 17},
		{"1.20.4", 17},
		{"1.20.5", 21},
		{"1.20.6", 21},
		{"1.21", 21},
		{"1.21.11", 21},
		{"21.1", 21},
		{"25.1", 21},
		{"26.1", 25},
		{"26.3", 25},
		{"27.1", 25},
	}

	for _, tt := range tests {
		if got := RequiredJavaVersion(tt.mc); got != tt.want {
			t.Errorf("RequiredJavaVersion(%q) = %d, want %d", tt.mc, got, tt.want)
		}
	}
}

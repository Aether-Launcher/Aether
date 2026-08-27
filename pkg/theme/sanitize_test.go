package theme

import "testing"

func TestSanitizeCSSManual(t *testing.T) {
	css := `
:root { --accent-color: #ff0000; --bg-color: #101010; }
.win-btn.close-btn { display: none; }
.titlebar .logo { content: "EvilBrand"; color: red; }
body { pointer-events: none; }
@import url("https://evil.com/track.css");
.card { border-radius: 12px; }
@media (max-width: 600px) {
  .win-btn { visibility: hidden; }
  .card { padding: 4px; }
}
`
	clean, warnings := SanitizeCSS(css)
	t.Logf("CLEAN:\n%s", clean)
	t.Logf("WARNINGS: %v", warnings)
	if contains(clean, "EvilBrand") {
		t.Fatal("brand content injection was not stripped")
	}
	if contains(clean, "evil.com") {
		t.Fatal("@import was not stripped")
	}
	if contains(clean, "close-btn") {
		t.Fatal("win-btn rule was not dropped")
	}
	if contains(clean, "pointer-events") {
		t.Fatal("pointer-events:none on body was not stripped")
	}
	if !contains(clean, "--accent-color: #ff0000") {
		t.Fatal("legit :root vars got mangled")
	}
	if !contains(clean, "border-radius: 12px") {
		t.Fatal("legit .card rule got mangled")
	}
	if contains(clean, "visibility: hidden") {
		t.Fatal("nested @media win-btn rule was not dropped")
	}
	if !contains(clean, "padding: 4px") {
		t.Fatal("legit nested @media rule got mangled")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

package keywordwatch

import "testing"

func TestNormalizeKeyword(t *testing.T) {
	display, norm, err := NormalizeKeyword("  Hello   World ")
	if err != nil {
		t.Fatal(err)
	}
	if display != "Hello World" || norm != "hello world" {
		t.Fatalf("got display=%q norm=%q", display, norm)
	}
	if _, _, err := NormalizeKeyword("a"); err == nil {
		t.Fatal("expected deny for common keyword")
	}
	if _, _, err := NormalizeKeyword("x"); err == nil {
		t.Fatal("expected too short")
	}
}

func TestContainsKeyword(t *testing.T) {
	html := `<p>Hello <a href="x">#RustLang</a> world</p>`
	if !ContainsKeyword(html, "rustlang") {
		t.Fatal("expected match")
	}
	if ContainsKeyword(html, "golang") {
		t.Fatal("unexpected match")
	}
}

func TestIsHashtagKeyword(t *testing.T) {
	if !IsHashtagKeyword("#Rust") || !IsHashtagKeyword("Rust") {
		t.Fatal("expected hashtag")
	}
	if IsHashtagKeyword("hello world") {
		t.Fatal("phrase is not hashtag")
	}
}

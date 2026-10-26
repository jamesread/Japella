package utils

import "testing"

func TestHTMLToText(t *testing.T) {
	in := `<p><span class="h-card"><a href="https://x/@me" class="u-url mention">@<span>me</span></a></span> nice &amp; tidy<br>line two</p><p>para two</p>`
	want := "@me nice & tidy\nline two\n\npara two"
	if got := HTMLToText(in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTextToSafeHTML(t *testing.T) {
	got := TextToSafeHTML("<img src=x onerror=alert(1)>\nhi & bye")
	want := "&lt;img src=x onerror=alert(1)&gt;<br>hi &amp; bye"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

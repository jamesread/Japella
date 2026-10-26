package languages

import "testing"

func TestBase(t *testing.T) {
	cases := map[string]string{
		"en": "en", " EN ": "en", "pt-BR": "pt", "zh_Hant": "zh", "fil": "fil",
		"": "", "und": "", "zxx": "", "qme": "", "english": "", "e": "", "12": "",
	}
	for in, want := range cases {
		if got := Base(in); got != want {
			t.Errorf("Base(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePreferred(t *testing.T) {
	got, err := ParsePreferred([]string{"de", "EN", "en-GB", " ", "pt-BR"})
	if err != nil {
		t.Fatal(err)
	}
	if Join(got) != "de,en,pt" {
		t.Fatalf("got %v", got)
	}
	if _, err := ParsePreferred([]string{"english"}); err == nil {
		t.Fatal("expected an error for a non-code")
	}
}

func TestFilter(t *testing.T) {
	if !NewFilter("").Allows("ja") {
		t.Fatal("no preferences: everything passes")
	}

	f := NewFilter("en,de")
	for tag, want := range map[string]bool{"en": true, "en-US": true, "DE": true, "ja": false, "": true, "und": true} {
		if got := f.Allows(tag); got != want {
			t.Errorf("Allows(%q) = %v, want %v", tag, got, want)
		}
	}
}

package validation

import "testing"

func TestIsValidEmail(t *testing.T) {
	cases := map[string]bool{
		"user@example.com":     true,
		"first.last@sub.co":    true,
		"invalid":              false,
		"missing-at.com":       false,
		"@missing-local.com":   false,
		"user@":                false,
		"user@domain":          false,
	}

	for email, want := range cases {
		if got := IsValidEmail(email); got != want {
			t.Errorf("IsValidEmail(%q) = %v, want %v", email, got, want)
		}
	}
}

func TestIsBlank(t *testing.T) {
	cases := map[string]bool{
		"":        true,
		"   ":     true,
		"\t\n":    true,
		"hello":   false,
		"  hi  ":  false,
	}

	for input, want := range cases {
		if got := IsBlank(input); got != want {
			t.Errorf("IsBlank(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestErrorsHasErrors(t *testing.T) {
	errs := Errors{}
	if errs.HasErrors() {
		t.Fatal("new Errors should have no errors")
	}

	errs.Add("title", "is required")
	if !errs.HasErrors() {
		t.Fatal("Errors with an entry should report HasErrors() == true")
	}
	if errs["title"] != "is required" {
		t.Errorf("expected error message 'is required', got %q", errs["title"])
	}
}

package jsonfmt

import "testing"

func TestSafeSnippet_Truncates(t *testing.T) {
	in := []byte("hello\x01world" + string(make([]byte, 2000)))
	out := SafeSnippet(in, 10)
	if len(out) > 20 {
		t.Fatalf("expected short output, got len %d", len(out))
	}
}

func TestPrettyJSON_Invalid(t *testing.T) {
	s := PrettyJSON([]byte(`not json`))
	if s == "" {
		t.Fatal("expected non-empty fallback")
	}
}

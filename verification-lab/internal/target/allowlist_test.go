package target

import "testing"

func TestBaseURLAllowlist(t *testing.T) {
	ok, err := BaseURL("http://127.0.0.1:8080/")
	if err != nil || ok != "http://127.0.0.1:8080" {
		t.Fatalf("ok=%q err=%v", ok, err)
	}
	_, err = BaseURL("http://evil.example/book")
	if err == nil {
		t.Fatal("expected reject external host")
	}
	_, err = BaseURL("https://127.0.0.1:8080")
	if err == nil {
		t.Fatal("expected reject https")
	}
}

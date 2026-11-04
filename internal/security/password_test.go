package security

import "testing"

func TestPasswordEncryptionRoundTrip(t *testing.T) {
	got, err := EncryptPassword("test-secret", "Manager123!")
	if err != nil {
		t.Fatal(err)
	}
	if got == "Manager123!" || got == "" {
		t.Fatal("password tidak boleh plaintext")
	}
	plain, err := DecryptPassword("test-secret", got)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "Manager123!" {
		t.Fatalf("got %q", plain)
	}
	if _, err := DecryptPassword("wrong-secret", got); err == nil {
		t.Fatal("secret salah seharusnya gagal")
	}
}

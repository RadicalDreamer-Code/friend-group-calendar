package users

import "testing"

func TestNormalizeUsername(t *testing.T) {
	valid := map[string]string{" Ada.L_1 ": "ada.l_1", "abc": "abc", "abcdefghijklmnopqrst": "abcdefghijklmnopqrst"}
	for in, want := range valid {
		got, err := NormalizeUsername(in)
		if err != nil || got != want {
			t.Errorf("NormalizeUsername(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "ab", "abcdefghijklmnopqrstu", "ada lovelace", "ada-l", "äda", "ada@x"} {
		if _, err := NormalizeUsername(in); err == nil {
			t.Errorf("NormalizeUsername(%q) should fail", in)
		}
	}
}

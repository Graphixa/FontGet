package fontkey

import "testing"

func TestKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"  Open Sans  ", "opensans"},
		{"Open-Sans", "opensans"},
		{"Open_Sans", "opensans"},
		{"OpenSans", "opensans"},
		{"", ""},
		{"already", "already"},
	}
	for _, tc := range cases {
		if got := Key(tc.in); got != tc.want {
			t.Fatalf("Key(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

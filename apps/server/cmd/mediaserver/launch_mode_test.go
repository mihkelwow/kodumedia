package main

import "testing"

func TestLaunchMode(t *testing.T) {
	tests := []struct {
		arguments []string
		want      string
	}{
		{want: "interactive"},
		{arguments: []string{"--startup"}, want: "windows-login"},
		{arguments: []string{" --STARTUP "}, want: "windows-login"},
		{arguments: []string{"--other"}, want: "interactive"},
	}
	for _, test := range tests {
		if got := launchMode(test.arguments); got != test.want {
			t.Fatalf("launchMode(%q) = %q, want %q", test.arguments, got, test.want)
		}
	}
}

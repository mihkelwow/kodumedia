package main

import "strings"

func launchMode(arguments []string) string {
	for _, argument := range arguments {
		if strings.EqualFold(strings.TrimSpace(argument), "--startup") {
			return "windows-login"
		}
	}
	return "interactive"
}

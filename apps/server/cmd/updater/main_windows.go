//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mihkelwow/kodumedia/internal/updater"
)

func main() {
	planPath := flag.String("plan", "", "absolute path to the KoduMedia update plan")
	flag.Parse()
	if *planPath == "" {
		fmt.Fprintln(os.Stderr, "missing --plan")
		os.Exit(2)
	}
	if err := updater.Run(*planPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

//go:build !windows

// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
package main

import (
	"fmt"
	"os"
)

func fatal(text string) { fmt.Fprintln(os.Stderr, text) }

func singleInstance() bool { return true }

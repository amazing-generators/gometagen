package main

import (
	"fmt"
	"os"
)

// // // // // // // // // //

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

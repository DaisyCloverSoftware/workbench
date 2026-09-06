// A zero-argument, read-only diagnostic. It cannot provision or start runners.
package main

import (
	"fmt"
	"github.com/DaisyCloverSoftware/workbench/internal/runnerdiag"
	"os"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "This diagnostic accepts no arguments")
		os.Exit(2)
	}
	result, err := runnerdiag.InspectJSON()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Windows runner diagnostic could not produce a bounded report")
		os.Exit(1)
	}
	fmt.Println(result)
}

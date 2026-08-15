//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("Workbench Dogfood is currently a native Windows dashboard. Use workbench-runner or workbench-server on this platform.")
}

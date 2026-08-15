//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("Workbench-Updater is available for Windows amd64. Cluster hosts use: workbench-runner update <check|apply>")
}

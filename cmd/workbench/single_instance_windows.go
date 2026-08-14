//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var workbenchSingleInstanceHandle uintptr

// The desktop and its embedded MCP server share one durable state file. A second
// native process would otherwise choose another free MCP port and could execute
// the same durable task concurrently. Acquire a per-user named mutex before
// main starts so only one desktop process can own that state at a time.
func init() {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	createMutex := kernel.NewProc("CreateMutexW")
	closeHandle := kernel.NewProc("CloseHandle")
	name, _ := syscall.UTF16PtrFromString(`Local\DaisyCloverSoftware.Workbench`)
	h, _, callErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		// Do not turn an unusual mutex creation failure into a startup regression;
		// the normal MCP bind checks still provide a second line of defence.
		return
	}
	workbenchSingleInstanceHandle = h
	if errno, ok := callErr.(syscall.Errno); !ok || errno != 183 { // ERROR_ALREADY_EXISTS
		return
	}

	user := syscall.NewLazyDLL("user32.dll")
	messageBox := user.NewProc("MessageBoxW")
	title, _ := syscall.UTF16PtrFromString("Workbench")
	text, _ := syscall.UTF16PtrFromString("Workbench is already running for this Windows user.")
	messageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x40) // MB_ICONINFORMATION
	closeHandle.Call(h)
	workbenchSingleInstanceHandle = 0
	os.Exit(0)
}

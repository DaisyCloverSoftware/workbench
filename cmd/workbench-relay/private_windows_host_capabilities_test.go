package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrivateCapabilitiesAdvertiseOnlyBoundedWindowsHostActions(t *testing.T) {
	b, err := privateChatCapabilitiesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var manifest privateChatCapabilities
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"list_windows_hosts", "run_windows_blender_version", "get_windows_host_job"} {
		if !containsString(manifest.ControlActions, action) {
			t.Fatalf("Windows host bridge action %q is not advertised", action)
		}
	}
	for _, unsafe := range []string{"run_windows_command", "run_host_command", "run_blender_command", "render_windows_blender"} {
		if containsString(manifest.ControlActions, unsafe) {
			t.Fatalf("unsafe/generic Windows action %q must not be advertised", unsafe)
		}
	}
	for _, want := range []string{"outbound-only", "no inbound Windows listener", "blender.exe --version", "no generic Windows command", "rendering is not enabled"} {
		if !strings.Contains(manifest.WindowsHostBridgePolicy, want) {
			t.Fatalf("Windows host bridge policy missing %q: %q", want, manifest.WindowsHostBridgePolicy)
		}
	}
	if !containsString(manifest.ChatGPTOwns, "bounded_windows_local_tool_execution") {
		t.Fatal("Windows local-tool execution ownership is not assigned to ChatGPT")
	}
}

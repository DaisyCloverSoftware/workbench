package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatCapabilitiesAreMachineReadableAndSecretFree(t *testing.T) {
	b, err := privateChatCapabilitiesJSON()
	if err != nil {
		t.Fatal(err)
	}
	if core.LooksSecret(string(b)) {
		t.Fatal("private relay capabilities must not contain secret-like material")
	}
	for _, literal := range []string{"relay/control/<id>.json", "relay/control-outbox/<id>.json", "relay/inbox/<id>.json", "relay/outbox/<id>.json", "relay/answers/<id>.json"} {
		if !strings.Contains(string(b), literal) {
			t.Fatalf("capability manifest should keep protocol path human-readable: missing %q in %s", literal, b)
		}
	}
	var manifest privateChatCapabilities
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Protocol != 1 || manifest.WorkbenchVersion != relayVersion || manifest.Transport != "private-git-relay" || manifest.PrimaryBrain != "chatgpt" {
		t.Fatalf("unexpected private relay manifest identity: %+v", manifest)
	}
	for _, want := range []string{"list_projects", "ensure_github_project", "read_file", "apply_patch", "run_safe_command", "search_memory", "save_context", "update_workbench"} {
		found := false
		for _, action := range manifest.ControlActions {
			if action == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("capability manifest missing control action %q", want)
		}
	}
	if manifest.ControlRequest != "relay/control/<id>.json" ||
		manifest.ControlResult != "relay/control-outbox/<id>.json" ||
		manifest.AutonomousRequest != "relay/inbox/<id>.json" ||
		manifest.AutonomousResult != "relay/outbox/<id>.json" ||
		manifest.AttentionAnswer != "relay/answers/<id>.json" {
		t.Fatalf("unexpected private relay protocol paths: %+v", manifest)
	}
	if !strings.Contains(manifest.ProjectReference, "exact opaque ref returned by list_projects") {
		t.Fatalf("capability manifest missing opaque project-ref rule: %q", manifest.ProjectReference)
	}
}

func TestPrivateChatCapabilitiesMatchImplementedControlActions(t *testing.T) {
	b, err := privateChatCapabilitiesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var manifest privateChatCapabilities
	if err := json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, action := range manifest.ControlActions {
		if isPrivateSafeHandsAction(action) {
			continue
		}
		switch action {
		case "save_memory", "search_memory", "save_context", "get_context", "update_workbench":
			continue
		default:
			t.Fatalf("capability manifest advertises unimplemented control action %q", action)
		}
	}
}

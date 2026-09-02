package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DaisyCloverSoftware/workbench/internal/core"
)

func TestPrivateChatCapabilitiesAreMachineReadableAndOpenClawDeniedByDefault(t *testing.T) {
	b,err:=privateChatCapabilitiesJSON(); if err!=nil{t.Fatal(err)}; text:=string(b); if core.LooksSecret(text){t.Fatal("manifest contains secret-like material")}
	for _,literal:=range []string{"relay/control/<id>.json","relay/control-outbox/<id>.json","inspect_machine","inspect_machine_batch","run_machine_command","run_operations_script","explicit_owner_request_only","owner_selected_openclaw_execution_only_no_automatic_routing",core.OpenClawExplicitAuthorizationPrefix,core.RelayOperationsIntentPrefix}{if !strings.Contains(text,literal){t.Fatalf("missing %q in %s",literal,text)}}
	for _,forbidden:=range []string{"optional_machine_side_autonomy_fallback","machine_operations_outside_the_direct_allowlist"}{if strings.Contains(text,forbidden){t.Fatalf("manifest still advertises automatic OpenClaw routing: %q",forbidden)}}
	var m privateChatCapabilities; if err:=json.Unmarshal(b,&m);err!=nil{t.Fatal(err)}
	if m.PrimaryBrain!="chatgpt"||m.OpenClawPolicy!="explicit_owner_request_only"{t.Fatalf("bad governance: %+v",m)}
	if m.AutonomousAuthorizationPrefix!=core.OpenClawExplicitAuthorizationPrefix||m.AutonomousOperationsPrefix!=core.RelayOperationsIntentPrefix{t.Fatalf("bad autonomous authorization contract: %+v",m)}
	for _,want:=range []string{"bounded_machine_inspection","bounded_machine_mutation","committed_operations_script_execution","subsequent_engineering_actions"}{if !containsString(m.ChatGPTOwns,want){t.Fatalf("ChatGPT ownership missing %q",want)}}
	for _,want:=range []string{"inspect_machine","inspect_machine_batch","run_machine_command","run_operations_script"}{if !containsString(m.ControlActions,want){t.Fatalf("direct action missing %q",want)}}
	if !strings.Contains(m.FreshChatBootstrap,"denied by default")||!strings.Contains(m.FreshChatBootstrap,"never authorize OpenClaw"){t.Fatalf("fresh bootstrap is ambiguous: %q",m.FreshChatBootstrap)}
}

func TestPrivateChatCapabilitiesMatchImplementedControlActions(t *testing.T){b,err:=privateChatCapabilitiesJSON();if err!=nil{t.Fatal(err)};var m privateChatCapabilities;if err:=json.Unmarshal(b,&m);err!=nil{t.Fatal(err)};for _,action:=range m.ControlActions{if isPrivateSafeHandsAction(action){continue};switch action{case "save_memory","search_memory","save_context","get_context","update_workbench":continue;default:t.Fatalf("manifest advertises unimplemented control action %q",action)}}}
func containsString(values []string,want string)bool{for _,v:=range values{if v==want{return true}};return false}

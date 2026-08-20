package main

import (
	"strings"
	"testing"
)

func TestHostJSONRejectsUnknownFields(t *testing.T) {
	response, code := processHostJSON(strings.NewReader(`{"action":"poll","unknown":true}`))
	if code != 2 || response.OK || !strings.Contains(response.Error, "unknown field") {
		t.Fatalf("code=%d response=%#v", code, response)
	}
}

func TestHostJSONPollRecordsHeartbeat(t *testing.T) {
	t.Setenv("WORKBENCH_HOST_BRIDGE_STATE_DIR", t.TempDir())
	request := `{"action":"poll","heartbeat":{"host_id":"windows_jsonhost","label":"JSON host","platform":"windows","arch":"amd64","capabilities":{"blender":{"installed":true,"version":"Blender 4.5.0"}}}}`
	response, code := processHostJSON(strings.NewReader(request))
	if code != 0 || !response.OK || response.Host == nil || response.Host.HostID != "windows_jsonhost" {
		t.Fatalf("code=%d response=%#v", code, response)
	}
}

func TestHostJSONRejectsMultipleValues(t *testing.T) {
	response, code := processHostJSON(strings.NewReader(`{"action":"poll"} {}`))
	if code != 2 || response.OK {
		t.Fatalf("code=%d response=%#v", code, response)
	}
}

package core

import (
	"reflect"
	"testing"
)

func TestHarnessProgressJSONUsesWorkProgressFieldNames(t *testing.T) {
	typ := reflect.TypeOf(HarnessProgress{})
	for _, field := range []string{"Kind", "Current", "Total", "Unit", "Phase", "Stage", "StageTotal"} {
		if _, ok := typ.FieldByName(field); !ok { t.Fatalf("missing HarnessProgress field %s", field) }
	}
}

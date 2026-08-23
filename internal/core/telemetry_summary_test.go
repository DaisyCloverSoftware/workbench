package core

import (
	"reflect"
	"testing"
)

func TestWorkProgressStoresInputsNotFabricatedPercentage(t *testing.T) {
	typ := reflect.TypeOf(WorkProgress{})
	if _, ok := typ.FieldByName("Percent"); ok {
		t.Fatal("WorkProgress must derive percentage from measured current/total rather than persist a guessed percent field")
	}
}

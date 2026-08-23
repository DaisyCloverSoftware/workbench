package core

import (
	"reflect"
	"testing"
)

func TestHarnessProgressCarriesNoAuthorityFields(t *testing.T) {
	typ := reflect.TypeOf(HarnessProgress{})
	for _, forbidden := range []string{"Command", "Path", "Token", "Secret", "URL", "Publish", "Deploy"} {
		if _, ok := typ.FieldByName(forbidden); ok { t.Fatalf("progress contract must not include authority field %s", forbidden) }
	}
}

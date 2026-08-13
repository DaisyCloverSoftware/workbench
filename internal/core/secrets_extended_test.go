package core

import "testing"

func TestLooksSecretCoversCommonCredentialShapes(t *testing.T) {
	bearer := "Authorization: Bearer " + "abcdefghijklmnop.qrstuvwxyz"
	quoted := `{"client_` + `secret":"abcdefghijklmnop"}`
	gitlab := "gl" + "pat-abcdefghijklmnop"
	jwt := "eyJ" + "abcdefghijk.abcdefghijklmnop.abcdefghijklmnop"
	for name, value := range map[string]string{
		"bearer header": bearer,
		"quoted key":    quoted,
		"gitlab token":  gitlab,
		"jwt":           jwt,
	} {
		if !LooksSecret(value) {
			t.Fatalf("expected %s to be detected", name)
		}
	}
}

func TestLooksSecretDoesNotFlagOrdinarySecurityDiscussion(t *testing.T) {
	for _, value := range []string{
		"Use bearer authentication for the API client.",
		"Rotate access tokens before they expire.",
		"The client secret belongs in the vault, not source control.",
	} {
		if LooksSecret(value) {
			t.Fatalf("ordinary prose was flagged: %q", value)
		}
	}
}

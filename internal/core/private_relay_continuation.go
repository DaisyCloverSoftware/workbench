package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const privateRelayContinuationProofPrefix = "[workbench:relay-continuation-proof:"

// SealPrivateRelayContinuationIntent binds a private-relay development handoff
// to the relay id, project and exact continuation body using the loopback MCP
// credential as an HMAC key. The credential itself never enters task state.
//
// For a deferred GitHub Actions handoff the wait locator is deliberately not
// signed: Workbench removes that first line when the dependency becomes
// terminal. The continuation body is what eventually reaches a coding worker,
// so that is the authority-bearing payload.
func SealPrivateRelayContinuationIntent(authValue, relayID, project, intent string) (string, error) {
	key := privateRelayContinuationKey(authValue)
	relayID = strings.TrimSpace(relayID)
	project = strings.TrimSpace(project)
	intent = strings.TrimSpace(intent)
	if len(key) == 0 || relayID == "" || project == "" || intent == "" {
		return "", errors.New("private relay continuation seal requires auth, relay id, project and intent")
	}
	body := privateRelayContinuationSignedBody(intent)
	if body == "" {
		return "", errors.New("private relay continuation body is empty")
	}
	proof := privateRelayContinuationMAC(key, relayID, project, body)
	return "[relay:" + relayID + "] " + intent + "\n" + privateRelayContinuationProofPrefix + relayID + ":" + proof + "]", nil
}

// ValidatePrivateRelayContinuationIntent proves that a non-operations
// chatgpt-mcp task was explicitly handed to the authenticated private relay.
// It returns only the original development intent, with transport correlation
// and proof material removed before a worker receives the task.
func ValidatePrivateRelayContinuationIntent(intent, project, mcpToken string) (string, bool) {
	intent = strings.TrimSpace(intent)
	project = strings.TrimSpace(project)
	key := privateRelayContinuationKey(mcpToken)
	if intent == "" || project == "" || len(key) == 0 {
		return "", false
	}

	lines := strings.Split(strings.ReplaceAll(intent, "\r\n", "\n"), "\n")
	if len(lines) < 2 {
		return "", false
	}
	proofLine := strings.TrimSpace(lines[len(lines)-1])
	if !strings.HasPrefix(proofLine, privateRelayContinuationProofPrefix) || !strings.HasSuffix(proofLine, "]") {
		return "", false
	}
	proofPayload := strings.TrimSuffix(strings.TrimPrefix(proofLine, privateRelayContinuationProofPrefix), "]")
	parts := strings.SplitN(proofPayload, ":", 2)
	if len(parts) != 2 || !validPrivateRelayID(parts[0]) {
		return "", false
	}
	relayID := parts[0]
	provided, err := hex.DecodeString(parts[1])
	if err != nil || len(provided) != sha256.Size {
		return "", false
	}

	body := strings.TrimSpace(strings.Join(lines[:len(lines)-1], "\n"))
	if strings.HasPrefix(body, "[relay:") {
		end := strings.Index(body, "] ")
		if end <= len("[relay:") || end > 96 || body[len("[relay:"):end] != relayID {
			return "", false
		}
		body = strings.TrimSpace(body[end+2:])
	}
	body = privateRelayContinuationSignedBody(body)
	if body == "" {
		return "", false
	}
	expected := privateRelayContinuationMACBytes(key, relayID, project, body)
	if !hmac.Equal(provided, expected) {
		return "", false
	}
	return body, true
}

func privateRelayContinuationSignedBody(intent string) string {
	intent = strings.TrimSpace(intent)
	lines := strings.SplitN(strings.ReplaceAll(intent, "\r\n", "\n"), "\n", 2)
	if len(lines) == 2 && strings.HasPrefix(strings.TrimSpace(lines[0]), githubActionsWaitPrefix) {
		return strings.TrimSpace(lines[1])
	}
	return intent
}

func privateRelayContinuationKey(authValue string) []byte {
	authValue = strings.TrimSpace(authValue)
	if strings.HasPrefix(strings.ToLower(authValue), "bearer ") {
		authValue = strings.TrimSpace(authValue[len("Bearer "):])
	}
	return []byte(authValue)
}

func privateRelayContinuationMAC(key []byte, relayID, project, body string) string {
	return hex.EncodeToString(privateRelayContinuationMACBytes(key, relayID, project, body))
}

func privateRelayContinuationMACBytes(key []byte, relayID, project, body string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("workbench-private-relay-continuation-v1\x00"))
	_, _ = mac.Write([]byte(relayID))
	_, _ = mac.Write([]byte("\x00"))
	_, _ = mac.Write([]byte(project))
	_, _ = mac.Write([]byte("\x00"))
	_, _ = mac.Write([]byte(strings.TrimSpace(body)))
	return mac.Sum(nil)
}

func validPrivateRelayID(id string) bool {
	if len(id) < 8 || len(id) > 80 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

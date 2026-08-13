package core

// claudeAllowedTools is deliberately narrower than granting unrestricted Bash.
// Claude Code can run these common local verification commands without pausing
// an unattended Workbench task for interactive approval. Anything outside this
// list stays under Claude's normal permission system and can be classified as a
// retryable worker limitation instead of becoming a human product decision.
func claudeAllowedTools() []string {
	return []string{
		"Bash(git status:*)",
		"Bash(git diff:*)",
		"Bash(git log:*)",
		"Bash(git show:*)",
		"Bash(git grep:*)",
		"Bash(git ls-files:*)",
		"Bash(go test:*)",
		"Bash(go vet:*)",
		"Bash(go build:*)",
		"Bash(gofmt:*)",
		"Bash(pytest:*)",
		"Bash(python -m pytest:*)",
		"Bash(cargo test:*)",
		"Bash(cargo check:*)",
		"Bash(dotnet test:*)",
		"Bash(npm test:*)",
		"Bash(npm run test:*)",
		"Bash(npm run build:*)",
		"Bash(npm run lint:*)",
		"Bash(pnpm test:*)",
		"Bash(pnpm run test:*)",
		"Bash(pnpm run build:*)",
		"Bash(pnpm run lint:*)",
		"Bash(yarn test:*)",
		"Bash(yarn run test:*)",
		"Bash(yarn run build:*)",
		"Bash(yarn run lint:*)",
		"Bash(mvn test:*)",
		"Bash(gradle test:*)",
		"Bash(./gradlew test:*)",
		"Bash(make test:*)",
	}
}

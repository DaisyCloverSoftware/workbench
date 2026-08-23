package core

import (
	"errors"
	"strings"
)

const ownerGatedProductMutationMessage = "owner-gated RUM/Rate Anything mutations are blocked from direct machine commands; use the reviewed committed operations path"

func validateOwnerGatedProductMutation(req MachineCommandRequest) error {
	program := strings.ToLower(strings.TrimSpace(req.Program))
	switch program {
	case "kubectl":
		if kubectlMutationTargetsOwnerGatedProduct(req.Args) {
			return errors.New(ownerGatedProductMutationMessage)
		}
	case "helm":
		if helmMutationTargetsOwnerGatedProduct(req.Args) {
			return errors.New(ownerGatedProductMutationMessage)
		}
	}
	return nil
}

func kubectlMutationTargetsOwnerGatedProduct(args []string) bool {
	if ownerGatedNamespace(machineNamespaceArg(args)) {
		return true
	}
	if len(args) == 0 {
		return false
	}

	start := 1
	switch strings.ToLower(args[0]) {
	case "rollout", "set":
		start = 2
	}
	if start > len(args) {
		return false
	}

	for _, arg := range args[start:] {
		if ownerGatedKubectlTarget(arg) {
			return true
		}
	}
	return false
}

func helmMutationTargetsOwnerGatedProduct(args []string) bool {
	if ownerGatedNamespace(machineNamespaceArg(args)) {
		return true
	}
	if len(args) < 2 {
		return false
	}

	command := strings.ToLower(strings.TrimSpace(args[0]))
	if command != "upgrade" && command != "install" && command != "rollback" {
		return false
	}

	release := strings.ToLower(strings.TrimSpace(args[1]))
	return ownerGatedName(release)
}

func machineNamespaceArg(args []string) string {
	for i, arg := range args {
		low := strings.ToLower(strings.TrimSpace(arg))
		switch {
		case low == "-n" || low == "--namespace":
			if i+1 < len(args) {
				return strings.ToLower(strings.TrimSpace(args[i+1]))
			}
		case strings.HasPrefix(low, "--namespace="):
			return strings.TrimSpace(strings.TrimPrefix(low, "--namespace="))
		case strings.HasPrefix(low, "-n="):
			return strings.TrimSpace(strings.TrimPrefix(low, "-n="))
		case strings.HasPrefix(low, "-n") && len(low) > 2:
			return strings.TrimSpace(strings.TrimPrefix(low, "-n"))
		}
	}
	return ""
}

func ownerGatedNamespace(namespace string) bool {
	return ownerGatedName(strings.ToLower(strings.TrimSpace(namespace)))
}

func ownerGatedKubectlTarget(arg string) bool {
	value := strings.ToLower(strings.TrimSpace(arg))
	if value == "" || strings.HasPrefix(value, "-") {
		return false
	}

	// Ignore ordinary label/annotation/container assignments such as app=rum or
	// php-fpm=ghcr.io/...; a Kubernetes resource target either has no assignment
	// or includes a resource/name slash before any assignment.
	if equal := strings.Index(value, "="); equal >= 0 {
		if slash := strings.Index(value, "/"); slash < 0 || slash > equal {
			return false
		}
		value = value[:equal]
	}

	for _, token := range strings.Split(value, ",") {
		token = strings.TrimSpace(token)
		if slash := strings.LastIndex(token, "/"); slash >= 0 {
			token = token[slash+1:]
		}
		if ownerGatedName(token) {
			return true
		}
	}
	return false
}

func ownerGatedName(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "rum" || strings.HasPrefix(value, "rum-")
}

package core

import (
	"context"
	"strings"
	"testing"
)

func TestRunMachineCommandBlocksOwnerGatedKubectlMutationBeforeExecution(t *testing.T) {
	_, err := RunMachineCommand(context.Background(), MachineCommandRequest{
		Program: "kubectl",
		Args: []string{
			"set", "image", "deployment/rum-api",
			"php-fpm=ghcr.io/daisycloversoftware/rum-api@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"-n", "rum-dev",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "owner-gated RUM/Rate Anything mutations") {
		t.Fatalf("expected protected RUM kubectl mutation to fail closed, got %v", err)
	}
}

func TestRunMachineCommandBlocksOwnerGatedHelmMutationBeforeExecution(t *testing.T) {
	_, err := RunMachineCommand(context.Background(), MachineCommandRequest{
		Program: "helm",
		Args:    []string{"rollback", "rum", "43", "-n", "rum-dev"},
	})
	if err == nil || !strings.Contains(err.Error(), "owner-gated RUM/Rate Anything mutations") {
		t.Fatalf("expected protected RUM helm mutation to fail closed, got %v", err)
	}
}

func TestOwnerGatedMutationDetectsProtectedResourceWithoutNamespace(t *testing.T) {
	req := MachineCommandRequest{
		Program: "kubectl",
		Args:    []string{"rollout", "restart", "deployment/rum-web"},
	}
	if err := validateOwnerGatedProductMutation(req); err == nil {
		t.Fatal("expected protected RUM resource target to be blocked")
	}
}

func TestOwnerGatedMutationDetectsProtectedNamespaceShortForms(t *testing.T) {
	for _, namespaceArg := range [][]string{
		{"-n", "rum-rate-anything-preview"},
		{"--namespace=rum-prod"},
		{"-n=rum-staging"},
		{"-nrum-dev"},
	} {
		args := []string{"scale", "deployment/example", "--replicas=1"}
		args = append(args, namespaceArg...)
		if err := validateOwnerGatedProductMutation(MachineCommandRequest{Program: "kubectl", Args: args}); err == nil {
			t.Fatalf("expected namespace args %v to be blocked", namespaceArg)
		}
	}
}

func TestOwnerGatedMutationDoesNotTreatAssignmentValueAsProtectedTarget(t *testing.T) {
	req := MachineCommandRequest{
		Program: "kubectl",
		Args:    []string{"label", "deployment/example", "app=rum", "-n", "example-dev"},
	}
	if err := validateOwnerGatedProductMutation(req); err != nil {
		t.Fatalf("expected unrelated label mutation to remain eligible, got %v", err)
	}
}

func TestOwnerGatedMutationLeavesUnrelatedMutationsEligible(t *testing.T) {
	requests := []MachineCommandRequest{
		{Program: "kubectl", Args: []string{"scale", "deployment/example", "--replicas=2", "-n", "example-dev"}},
		{Program: "helm", Args: []string{"upgrade", "example", "./chart", "-n", "example-dev"}},
	}
	for _, req := range requests {
		if err := validateOwnerGatedProductMutation(req); err != nil {
			t.Fatalf("expected unrelated mutation %s %v to remain eligible, got %v", req.Program, req.Args, err)
		}
	}
}

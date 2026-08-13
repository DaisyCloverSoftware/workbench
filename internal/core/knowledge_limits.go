package core

import "fmt"

const (
	knowledgeTitleLimit   = 240
	knowledgeContentLimit = 32768
	knowledgeSourceLimit  = 1024
	knowledgeTagCount     = 16
	knowledgeTagLimit     = 96
	knowledgeProjectLimit = 4096
	contextObjectiveLimit = 4096
	contextStateLimit     = 16384
	contextNextLimit      = 4096
	contextListCount      = 24
	contextListItemLimit  = 2048
)

func checkKnowledgeLimits(item KnowledgeItem) error {
	if len(item.Project) > knowledgeProjectLimit {
		return fmt.Errorf("knowledge project is too large")
	}
	if len(item.Title) > knowledgeTitleLimit {
		return fmt.Errorf("knowledge title is too large")
	}
	if len(item.Content) > knowledgeContentLimit {
		return fmt.Errorf("knowledge content is too large")
	}
	if len(item.Source) > knowledgeSourceLimit {
		return fmt.Errorf("knowledge source is too large")
	}
	if len(item.Tags) > knowledgeTagCount {
		return fmt.Errorf("knowledge has too many tags")
	}
	for _, tag := range item.Tags {
		if len(tag) > knowledgeTagLimit {
			return fmt.Errorf("knowledge tag is too large")
		}
	}
	return nil
}

func checkContextLimits(c ContextCapsule) error {
	if len(c.Project) > knowledgeProjectLimit || len(c.Objective) > contextObjectiveLimit || len(c.State) > contextStateLimit || len(c.NextAction) > contextNextLimit {
		return fmt.Errorf("context field is too large")
	}
	groups := [][]string{c.Decisions, c.Constraints, c.References, c.OpenThreads}
	for _, values := range groups {
		if len(values) > contextListCount {
			return fmt.Errorf("context list has too many items")
		}
		for _, value := range values {
			if len(value) > contextListItemLimit {
				return fmt.Errorf("context list item is too large")
			}
		}
	}
	return nil
}

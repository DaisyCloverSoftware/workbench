package core

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

// DecisionRecord is the privacy-minimal searchable decision projection. It can
// represent an explicit durable decision memory or a decision captured in a
// continuation capsule without exposing the host-specific project path.
type DecisionRecord struct {
	ID         string         `json:"id"`
	Scope      KnowledgeScope `json:"scope"`
	Title      string         `json:"title"`
	Decision   string         `json:"decision"`
	Tags       []string       `json:"tags,omitempty"`
	SourceType string         `json:"source_type"`
	ContextID  string         `json:"context_id,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type KnowledgeGraphNode struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Label     string         `json:"label"`
	Kind      KnowledgeKind  `json:"kind,omitempty"`
	Scope     KnowledgeScope `json:"scope,omitempty"`
	Content   string         `json:"content,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	UpdatedAt time.Time      `json:"updated_at,omitempty"`
}

type KnowledgeGraphEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

type KnowledgeGraph struct {
	Query           string               `json:"query,omitempty"`
	ProjectSelected bool                 `json:"project_selected"`
	Nodes           []KnowledgeGraphNode `json:"nodes"`
	Edges           []KnowledgeGraphEdge `json:"edges"`
}

// SearchKnowledgeKinds is SearchKnowledge with an optional kind filter. It is
// kept in core so MCP, private relay and future desktop search all share exactly
// the same scope and relevance rules.
func SearchKnowledgeKinds(project, query string, kinds []KnowledgeKind, limit int) ([]KnowledgeItem, error) {
	knowledgeMu.RLock()
	defer knowledgeMu.RUnlock()

	st, err := loadKnowledgeState()
	if err != nil {
		return nil, err
	}
	project = strings.TrimSpace(project)
	terms := searchTerms(query)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	kindSet := make(map[KnowledgeKind]bool, len(kinds))
	for _, kind := range kinds {
		if kind != "" {
			kindSet[kind] = true
		}
	}
	type scored struct {
		item  KnowledgeItem
		score int
	}
	found := make([]scored, 0, len(st.Items))
	for _, item := range st.Items {
		if item.Scope == ScopeProject && item.Project != project {
			continue
		}
		if len(kindSet) > 0 && !kindSet[item.Kind] {
			continue
		}
		score := knowledgeSearchScore(item.Title, item.Content, item.Tags, terms)
		if len(terms) == 0 {
			score = 1
		}
		if item.Project == project && project != "" {
			score++
		}
		if score > 0 {
			found = append(found, scored{item: item, score: score})
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].score != found[j].score {
			return found[i].score > found[j].score
		}
		return found[i].item.UpdatedAt.After(found[j].item.UpdatedAt)
	})
	out := make([]KnowledgeItem, 0, minInt(limit, len(found)))
	for i := 0; i < len(found) && i < limit; i++ {
		out = append(out, found[i].item)
	}
	return out, nil
}

func knowledgeSearchScore(title, content string, tags, terms []string) int {
	titleLower := strings.ToLower(title)
	hay := strings.ToLower(title + "\n" + content + "\n" + strings.Join(tags, " "))
	score := 0
	for _, term := range terms {
		if strings.Contains(hay, term) {
			score += 3
		}
		if strings.Contains(titleLower, term) {
			score += 3
		}
	}
	return score
}

// SearchDecisions searches both explicit decision memories and decisions carried
// forward in context capsules. Repeated capsule decisions are de-duplicated by
// normalized decision text; an explicit durable memory wins over a capsule copy.
func SearchDecisions(project, query string, limit int) ([]DecisionRecord, error) {
	knowledgeMu.RLock()
	defer knowledgeMu.RUnlock()

	st, err := loadKnowledgeState()
	if err != nil {
		return nil, err
	}
	project = strings.TrimSpace(project)
	terms := searchTerms(query)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	type scored struct {
		record DecisionRecord
		score  int
		memory bool
	}
	byDecision := map[string]scored{}
	add := func(key string, candidate scored) {
		if key == "" || candidate.score <= 0 {
			return
		}
		old, exists := byDecision[key]
		if !exists || candidate.score > old.score || (candidate.score == old.score && candidate.memory && !old.memory) || (candidate.score == old.score && candidate.memory == old.memory && candidate.record.UpdatedAt.After(old.record.UpdatedAt)) {
			byDecision[key] = candidate
		}
	}

	for _, item := range st.Items {
		if item.Kind != KindDecision || (item.Scope == ScopeProject && item.Project != project) {
			continue
		}
		score := knowledgeSearchScore(item.Title, item.Content, item.Tags, terms)
		if len(terms) == 0 {
			score = 1
		}
		if item.Scope == ScopeProject && item.Project == project && project != "" {
			score++
		}
		add(normalizedDecisionKey(item.Content), scored{
			record: DecisionRecord{ID: item.ID, Scope: item.Scope, Title: item.Title, Decision: item.Content, Tags: append([]string(nil), item.Tags...), SourceType: "memory", UpdatedAt: item.UpdatedAt},
			score:  score,
			memory: true,
		})
	}
	if project != "" {
		for _, capsule := range st.Capsules {
			if capsule.Project != project {
				continue
			}
			for _, decision := range capsule.Decisions {
				decision = strings.TrimSpace(decision)
				if decision == "" {
					continue
				}
			score := knowledgeSearchScore(capsule.Objective, decision, nil, terms)
				if len(terms) == 0 {
					score = 1
				}
				score++
				add(normalizedDecisionKey(decision), scored{
					record: DecisionRecord{ID: contextDecisionID(project, decision), Scope: ScopeProject, Title: capsule.Objective, Decision: decision, SourceType: "context", ContextID: capsule.ID, UpdatedAt: capsule.UpdatedAt},
					score:  score,
				})
			}
		}
	}

	found := make([]scored, 0, len(byDecision))
	for _, candidate := range byDecision {
		found = append(found, candidate)
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].score != found[j].score {
			return found[i].score > found[j].score
		}
		return found[i].record.UpdatedAt.After(found[j].record.UpdatedAt)
	})
	out := make([]DecisionRecord, 0, minInt(limit, len(found)))
	for i := 0; i < len(found) && i < limit; i++ {
		out = append(out, found[i].record)
	}
	return out, nil
}

func normalizedDecisionKey(decision string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(decision))), " ")
}

func contextDecisionID(project, decision string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(project) + "\n" + normalizedDecisionKey(decision)))
	return "ctxdec-" + hex.EncodeToString(sum[:8])
}

// BuildKnowledgeGraph returns a compact derived graph; it does not create a
// second knowledge database. Memory IDs remain the authoritative durable nodes,
// while project/scope/tag nodes provide traversal edges. Capsule-only decisions
// appear as derived decision nodes so important continuation decisions remain
// discoverable until they are promoted to durable decision memory.
func BuildKnowledgeGraph(project, query string, limit int) (KnowledgeGraph, error) {
	project = strings.TrimSpace(project)
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	items, err := SearchKnowledgeKinds(project, query, nil, limit)
	if err != nil {
		return KnowledgeGraph{}, err
	}
	decisions, err := SearchDecisions(project, query, limit)
	if err != nil {
		return KnowledgeGraph{}, err
	}
	graph := KnowledgeGraph{Query: strings.TrimSpace(query), ProjectSelected: project != "", Nodes: []KnowledgeGraphNode{}, Edges: []KnowledgeGraphEdge{}}
	nodeSeen := map[string]bool{}
	edgeSeen := map[string]bool{}
	addNode := func(node KnowledgeGraphNode) {
		if node.ID == "" || nodeSeen[node.ID] {
			return
		}
		nodeSeen[node.ID] = true
		graph.Nodes = append(graph.Nodes, node)
	}
	addEdge := func(edge KnowledgeGraphEdge) {
		if edge.From == "" || edge.To == "" || edge.Relation == "" {
			return
		}
		key := edge.From + "\x00" + edge.Relation + "\x00" + edge.To
		if edgeSeen[key] {
			return
		}
		edgeSeen[key] = true
		graph.Edges = append(graph.Edges, edge)
	}

	projectNode := ""
	if project != "" {
		projectNode = opaqueProjectKnowledgeNodeID(project)
		addNode(KnowledgeGraphNode{ID: projectNode, Type: "project", Label: "Project knowledge", Scope: ScopeProject})
	}
	globalNode := "scope:global"
	memoryIDs := map[string]bool{}
	for _, item := range items {
		memoryIDs[item.ID] = true
		addNode(KnowledgeGraphNode{ID: item.ID, Type: "memory", Label: item.Title, Kind: item.Kind, Scope: item.Scope, Content: boundedGraphContent(item.Content), Tags: append([]string(nil), item.Tags...), UpdatedAt: item.UpdatedAt})
		if item.Scope == ScopeGlobal {
			addNode(KnowledgeGraphNode{ID: globalNode, Type: "scope", Label: "Global knowledge", Scope: ScopeGlobal})
			addEdge(KnowledgeGraphEdge{From: globalNode, To: item.ID, Relation: "contains"})
		} else if projectNode != "" {
			addEdge(KnowledgeGraphEdge{From: projectNode, To: item.ID, Relation: "contains"})
		}
		for _, tag := range item.Tags {
			tagID := graphTagID(tag)
			addNode(KnowledgeGraphNode{ID: tagID, Type: "tag", Label: tag})
			addEdge(KnowledgeGraphEdge{From: item.ID, To: tagID, Relation: "tagged"})
		}
	}
	for _, decision := range decisions {
		if decision.SourceType == "memory" && memoryIDs[decision.ID] {
			continue
		}
		addNode(KnowledgeGraphNode{ID: decision.ID, Type: "decision", Label: decision.Title, Kind: KindDecision, Scope: decision.Scope, Content: boundedGraphContent(decision.Decision), Tags: append([]string(nil), decision.Tags...), UpdatedAt: decision.UpdatedAt})
		if decision.Scope == ScopeGlobal {
			addNode(KnowledgeGraphNode{ID: globalNode, Type: "scope", Label: "Global knowledge", Scope: ScopeGlobal})
			addEdge(KnowledgeGraphEdge{From: globalNode, To: decision.ID, Relation: "contains"})
		} else if projectNode != "" {
			addEdge(KnowledgeGraphEdge{From: projectNode, To: decision.ID, Relation: "contains"})
		}
		for _, tag := range decision.Tags {
			tagID := graphTagID(tag)
			addNode(KnowledgeGraphNode{ID: tagID, Type: "tag", Label: tag})
			addEdge(KnowledgeGraphEdge{From: decision.ID, To: tagID, Relation: "tagged"})
		}
	}

	sort.SliceStable(graph.Nodes, func(i, j int) bool {
		if graph.Nodes[i].Type != graph.Nodes[j].Type {
			return graph.Nodes[i].Type < graph.Nodes[j].Type
		}
		return graph.Nodes[i].ID < graph.Nodes[j].ID
	})
	sort.SliceStable(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From != graph.Edges[j].From {
			return graph.Edges[i].From < graph.Edges[j].From
		}
		if graph.Edges[i].Relation != graph.Edges[j].Relation {
			return graph.Edges[i].Relation < graph.Edges[j].Relation
		}
		return graph.Edges[i].To < graph.Edges[j].To
	})
	return graph, nil
}

func opaqueProjectKnowledgeNodeID(project string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(project)))
	return "project-knowledge-" + hex.EncodeToString(sum[:8])
}

func graphTagID(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	sum := sha256.Sum256([]byte(tag))
	return "tag-" + hex.EncodeToString(sum[:8])
}

func boundedGraphContent(content string) string {
	content = strings.TrimSpace(content)
	const max = 480
	if len(content) <= max {
		return content
	}
	return strings.TrimSpace(content[:max]) + "…"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package model

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
)

// Each report owns its visited pairs. Candidate scores are shared within one
// public comparison, but scoring never marks a pair visited in the report.
type schemaComparison struct {
	visited map[schemaComparisonPair]struct{}
	scores  map[schemaScorePair]schemaScore
	scoring *schemaScoring
}

// Active scores belong to one traversal. A recursion cut must not publish a
// context-dependent partial score for another candidate to reuse.
type schemaScoring struct {
	active map[schemaScorePair]struct{}
	cycles uint64
}

type schemaComparisonPair struct {
	left, right *yaml.Node
}

type schemaScoreProxy struct {
	node      *yaml.Node
	index     *index.SpecIndex
	reference string
	isRef     bool
}

type schemaScorePair struct {
	left, right schemaScoreProxy
}

type schemaScore struct {
	total, breaking int
}

func schemaScoreIdentity(proxy *base.SchemaProxy) schemaScoreProxy {
	if proxy == nil {
		return schemaScoreProxy{}
	}
	return schemaScoreProxy{schemaDefinitionNode(proxy), proxy.GetIndex(), proxy.GetReference(), proxy.IsReference()}
}

func (c *schemaComparison) score(left, right *base.SchemaProxy) schemaScore {
	pair := schemaScorePair{schemaScoreIdentity(left), schemaScoreIdentity(right)}
	if score, exists := c.scores[pair]; exists {
		return score
	}
	if c.scoring == nil {
		c.scoring = &schemaScoring{active: make(map[schemaScorePair]struct{})}
	}
	if _, active := c.scoring.active[pair]; active {
		c.scoring.cycles++
		return schemaScore{}
	}
	c.scoring.active[pair] = struct{}{}
	defer delete(c.scoring.active, pair)
	cycles := c.scoring.cycles
	if c.scores == nil {
		c.scores = make(map[schemaScorePair]schemaScore)
	}
	// Each candidate must see a complete comparison, independent of earlier
	// candidates and report placement. Keep only its counts, not its report nodes.
	candidate := schemaComparison{scores: c.scores, scoring: c.scoring}
	var score schemaScore
	if changes := candidate.compareSchemas(left, right); changes != nil {
		score = schemaScore{changes.TotalChanges(), changes.TotalBreakingChanges()}
	}
	if c.scoring.cycles == cycles {
		c.scores[pair] = score
	}
	return score
}

func (c *schemaComparison) visit(left, right *base.SchemaProxy) bool {
	pair := schemaComparisonPair{schemaDefinitionNode(left), schemaDefinitionNode(right)}
	if _, exists := c.visited[pair]; exists {
		return false
	}
	if c.visited == nil {
		c.visited = make(map[schemaComparisonPair]struct{})
	}
	c.visited[pair] = struct{}{}
	return true
}

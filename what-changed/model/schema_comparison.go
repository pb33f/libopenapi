package model

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"runtime"
	"sync"
)

// Each report owns its visited pairs. Candidate scores are shared within one
// public comparison, but scoring never marks a pair visited in the report.
type schemaComparison struct {
	visited   map[schemaComparisonPair]struct{}
	inherited map[schemaComparisonPair]struct{}
	previous  map[schemaComparisonPair]struct{}
	scores    *schemaScores
	scoring   *schemaScoring
	serial    bool
}

type schemaScores struct {
	sync.RWMutex
	values map[schemaScorePair]schemaScore
}

// branch compares independent siblings with bounded workers. Forks read the
// parent's fixed visited set and keep private deltas. Reconcile in report order;
// replay an overlapping fork serially to retain one stable location per pair.
// Each worker remembers its earlier siblings so shared subtrees are expanded
// at most once per worker, rather than once per sibling.
func (c *schemaComparison) branch(count int, task func(int, *schemaComparison)) {
	if c.serial || count < 2 || runtime.GOMAXPROCS(0) == 1 {
		for i := range count {
			task(i, c)
		}
		return
	}
	if c.scores == nil {
		c.scores = new(schemaScores)
	}
	deltas := make([]map[schemaComparisonPair]struct{}, count)
	workers := min(count, runtime.GOMAXPROCS(0), 4)
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fork := schemaComparison{inherited: c.visited, scores: c.scores, serial: true}
			for i := worker; i < count; i += workers {
				fork.visited = nil
				task(i, &fork)
				deltas[i] = fork.visited
				if len(fork.visited) > 0 {
					if fork.previous == nil {
						fork.previous = make(map[schemaComparisonPair]struct{})
					}
					for pair := range fork.visited {
						fork.previous[pair] = struct{}{}
					}
				}
			}
		}()
	}
	wg.Wait()
	for i, delta := range deltas {
		overlap := false
		for pair := range delta {
			if _, exists := c.visited[pair]; exists {
				overlap = true
				break
			}
		}
		if overlap {
			c.serial = true
			task(i, c)
			c.serial = false
			continue
		}
		if len(delta) > 0 {
			if c.visited == nil {
				c.visited = delta
			} else {
				for pair := range delta {
					c.visited[pair] = struct{}{}
				}
			}
		}
	}
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

type schemaCompositionTask struct {
	left, right []low.ValueReference[*base.SchemaProxy]
	label       string
	result      *[]*SchemaChanges
	changes     []*Change
}

func (c *schemaComparison) compareCompositions(tasks []schemaCompositionTask, changes *[]*Change) {
	active := tasks[:0]
	for _, task := range tasks {
		if len(task.left) > 0 || len(task.right) > 0 {
			active = append(active, task)
		}
	}
	c.branch(len(active), func(i int, branch *schemaComparison) {
		task := &active[i]
		*task.result = nil
		task.changes = nil
		branch.extractSchemaChanges(task.left, task.right, task.label, task.result, &task.changes)
	})
	for _, task := range active {
		*changes = append(*changes, task.changes...)
	}
}

func schemaScoreIdentity(proxy *base.SchemaProxy) schemaScoreProxy {
	if proxy == nil {
		return schemaScoreProxy{}
	}
	return schemaScoreProxy{schemaDefinitionNode(proxy), proxy.GetIndex(), proxy.GetReference(), proxy.IsReference()}
}

func (c *schemaComparison) score(left, right *base.SchemaProxy) schemaScore {
	pair := schemaScorePair{schemaScoreIdentity(left), schemaScoreIdentity(right)}
	if c.scores != nil {
		c.scores.RLock()
		score, exists := c.scores.values[pair]
		c.scores.RUnlock()
		if exists {
			return score
		}
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
		c.scores = new(schemaScores)
	}
	// Each candidate must see a complete comparison, independent of earlier
	// candidates and report placement. Keep only its counts, not its report nodes.
	candidate := schemaComparison{scores: c.scores, scoring: c.scoring, serial: true}
	var score schemaScore
	if changes := candidate.compareSchemas(left, right); changes != nil {
		score = schemaScore{changes.TotalChanges(), changes.TotalBreakingChanges()}
	}
	if c.scoring.cycles == cycles {
		c.scores.Lock()
		if c.scores.values == nil {
			c.scores.values = make(map[schemaScorePair]schemaScore)
		}
		c.scores.values[pair] = score
		c.scores.Unlock()
	}
	return score
}

func (c *schemaComparison) visit(left, right *base.SchemaProxy) bool {
	pair := schemaComparisonPair{schemaDefinitionNode(left), schemaDefinitionNode(right)}
	if _, exists := c.visited[pair]; exists {
		return false
	}
	if _, exists := c.inherited[pair]; exists {
		return false
	}
	if _, exists := c.previous[pair]; exists {
		return false
	}
	if c.visited == nil {
		c.visited = make(map[schemaComparisonPair]struct{})
	}
	c.visited[pair] = struct{}{}
	return true
}

package index

import "strings"

// CircularReferenceResult contains a circular reference found when traversing the graph.
type CircularReferenceResult struct {
	Journey             []*Reference
	Start               *Reference
	LoopIndex           int
	LoopPoint           *Reference
	IsPolymorphicResult bool // if this result comes from a polymorphic loop.
}

func (c *CircularReferenceResult) GenerateJourneyPath() string {
	buf := strings.Builder{}
	for i, ref := range c.Journey {
		buf.WriteString(ref.Name)
		if i+1 < len(c.Journey) {
			buf.WriteString(" -> ")
		}
	}
	return buf.String()
}

package index

import "strings"

// CircularReferenceResult contains a circular reference found when traversing the graph.
type CircularReferenceResult struct {
	Journey             []*Reference
	Start               *Reference
	LoopIndex           int
	LoopPoint           *Reference
	IsPolymorphicResult bool // if this result comes from a polymorphic loop.
	IsInfiniteLoop      bool // if all the definitions in the reference loop are marked as required, this is an infinite circular reference, thus is not allowed.
}

func (c *CircularReferenceResult) GenerateJourneyPath() string {
	buf := strings.Builder{}
	for i, ref := range c.Journey {
		if i > 0 {
			buf.WriteString(" -> ")
		}

		buf.WriteString(ref.Name)
		// buf.WriteString(" (")
		// buf.WriteString(ref.Definition)
		// buf.WriteString(")")
	}

	return buf.String()
}

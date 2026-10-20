package index

import "github.com/pb33f/go-yaml"

// frozen origin lookup from libopenapi cbdd043c.
func (r *Rolodex) findNodeOriginWithValueReference(key, value, refNode *yaml.Node, refValue string) *NodeOrigin {
	if key == nil {
		return nil
	}
	keyOrigin := r.FindNodeOrigin(key)
	var valueOrigin *NodeOrigin
	var valueHash string
	if value != nil {
		if keyOrigin != nil && keyOrigin.AbsoluteLocation == r.GetRootIndex().specAbsolutePath {
			valueOrigin = r.GetRootIndex().FindNodeOrigin(value)
			valueHash = HashNode(value)
			if refNode != nil && refValue != "" {
				return keyOrigin
			}
			origin, done := checkOrigin(originCheck{
				valueOrigin: valueOrigin,
				valueHash:   valueHash,
				keyOrigin:   keyOrigin,
				value:       value,
				rolodex:     r,
				ref:         refValue,
				refNode:     refNode,
			})
			if done {
				return origin
			} else {
				return nil
			}
		} else {
			// the value is not in the root index, so we need to search all indexes
			for i := range r.indexes {
				idx := r.indexes[i]
				if keyOrigin == nil {
					keyOrigin = idx.FindNodeOrigin(key)
				}
				n := idx.FindNodeOrigin(value)
				if n != nil {
					if refNode != nil && refValue != "" {
						return keyOrigin
					}

					valueHash = HashNode(value)
					nHash := HashNode(n.Node)

					if valueHash == nHash {
						if keyOrigin.AbsoluteLocation != n.AbsoluteLocation {
							if refNode == nil && refValue == "" {
								keyOrigin.AbsoluteLocationValue = n.AbsoluteLocation
								keyOrigin.LineValue = n.Line
								keyOrigin.ColumnValue = n.Column
								keyOrigin.ValueNode = n.Node
							}
						}
						return keyOrigin
					}
				}
			}
		}
	}
	return keyOrigin
}

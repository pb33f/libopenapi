package orderedmap

import (
	"fmt"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/nodes"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

type Marshaler interface {
	MarshalYAML() (interface{}, error)
}

type NodeBuilder interface {
	AddYAMLNode(parent *yaml.Node, entry *nodes.NodeEntry) *yaml.Node
}

type MapToYamlNoder interface {
	ToYamlNode(n NodeBuilder, l any) *yaml.Node
}

type HasKeyNode interface {
	GetKeyNode() *yaml.Node
}

type HasValueNode interface {
	GetValueNode() *yaml.Node
}

type HasValueUntyped interface {
	GetValueUntyped() any
}

type FindValueUntyped interface {
	FindValueUntyped(k string) any
}

func (o *Map[K, V]) ToYamlNode(n NodeBuilder, l any) *yaml.Node {
	p := utils.CreateEmptyMapNode()

	var vn *yaml.Node

	i := 99999
	if l != nil {
		if hvn, ok := l.(HasValueNode); ok {
			vn = hvn.GetValueNode()
			if vn != nil && len(vn.Content) > 0 {
				i = vn.Content[0].Line
			}
		}
	}

	for pair := First(o); pair != nil; pair = pair.Next() {
		var k any = pair.Key()
		if m, ok := k.(Marshaler); ok { // TODO marshal inline?
			k, _ = m.MarshalYAML()
		}

		var y any
		y, ok := k.(yaml.Node)
		if !ok {
			y, ok = k.(*yaml.Node)
		}
		if ok {
			b, _ := yaml.Marshal(y)
			k = strings.TrimSpace(string(b))
		}

		ks := k.(string)

		var keyStyle yaml.Style
		keyNode := findKeyNode(ks, vn)
		if keyNode != nil {
			keyStyle = keyNode.Style
		}

		var lv any
		if l != nil {
			if hvut, ok := l.(HasValueUntyped); ok {
				vut := hvut.GetValueUntyped()
				if m, ok := vut.(FindValueUntyped); ok {
					lv = m.FindValueUntyped(ks)
				}
			}
		}

		n.AddYAMLNode(p, &nodes.NodeEntry{
			Tag:      ks,
			Key:      ks,
			Line:     i,
			Value:    pair.Value(),
			KeyStyle: keyStyle,
			LowValue: lv,
		})
		i++
	}

	return p
}

func findKeyNode(key string, m *yaml.Node) *yaml.Node {
	if m == nil {
		return nil
	}

	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i]
		}
	}
	return nil
}

func (o *Map[K, V]) FindValueUntyped(key string) any {
	for pair := First(o); pair != nil; pair = pair.Next() {
		var k any = pair.Key()
		if hvut, ok := k.(HasValueUntyped); ok {
			if fmt.Sprintf("%v", hvut.GetValueUntyped()) == key {
				return pair.Value()
			}
		}
		if fmt.Sprintf("%v", k) == key {
			return pair.Value()
		}
	}

	return nil
}

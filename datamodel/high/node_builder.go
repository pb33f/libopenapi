// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package high

import (
    "fmt"
    "github.com/pb33f/libopenapi/datamodel/low"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
    "reflect"
    "sort"
    "strconv"
    "strings"
    "unicode"
)

// NodeEntry represents a single node used by NodeBuilder.
type NodeEntry struct {
    Tag         string
    Key         string
    Value       any
    StringValue string
    Line        int
    RenderZero  bool
}

// NodeBuilder is a structure used by libopenapi high-level objects, to render themselves back to YAML.
// this allows high-level objects to be 'mutable' because all changes will be rendered out.
type NodeBuilder struct {
    Nodes   []*NodeEntry
    High    any
    Low     any
    Resolve bool // If set to true, all references will be rendered inline
}

const renderZero = "renderZero"

// NewNodeBuilder will create a new NodeBuilder instance, this is the only way to create a NodeBuilder.
// The function accepts a high level object and a low level object (need to be siblings/same type).
//
// Using reflection, a map of every field in the high level object is created, ready to be rendered.
func NewNodeBuilder(high any, low any) *NodeBuilder {
    // create a new node builder
    nb := new(NodeBuilder)
    nb.High = high
    if low != nil {
        nb.Low = low
    }

    // extract fields from the high level object and add them into our node builder.
    // this will allow us to extract the line numbers from the low level object as well.
    v := reflect.ValueOf(high).Elem()
    num := v.NumField()
    for i := 0; i < num; i++ {
        nb.add(v.Type().Field(i).Name, i)
    }
    return nb
}

func (n *NodeBuilder) add(key string, i int) {

    // only operate on exported fields.
    if unicode.IsLower(rune(key[0])) {
        return
    }

    // if the key is 'Extensions' then we need to extract the keys from the map
    // and add them to the node builder.
    if key == "Extensions" {
        extensions := reflect.ValueOf(n.High).Elem().FieldByName(key)
        for b, e := range extensions.MapKeys() {
            v := extensions.MapIndex(e)

            extKey := e.String()
            extValue := v.Interface()
            nodeEntry := &NodeEntry{Tag: extKey, Key: extKey, Value: extValue, Line: 9999 + b}

            if n.Low != nil && !reflect.ValueOf(n.Low).IsZero() {
                fieldValue := reflect.ValueOf(n.Low).Elem().FieldByName("Extensions")
                f := fieldValue.Interface()
                value := reflect.ValueOf(f)
                switch value.Kind() {
                case reflect.Map:
                    if j, ok := n.Low.(low.HasExtensionsUntyped); ok {
                        originalExtensions := j.GetExtensions()
                        u := 0
                        for k := range originalExtensions {
                            if k.Value == extKey {
                                if originalExtensions[k].ValueNode.Line != 0 {
                                    nodeEntry.Line = originalExtensions[k].ValueNode.Line + u
                                } else {
                                    nodeEntry.Line = 999999 + b + u
                                }
                            }
                            u++
                        }
                    }
                }
            }
            n.Nodes = append(n.Nodes, nodeEntry)
        }
        // done, extensions are handled separately.
        return
    }

    // find the field with the tag supplied.
    field, _ := reflect.TypeOf(n.High).Elem().FieldByName(key)
    tag := string(field.Tag.Get("yaml"))
    tagName := strings.Split(tag, ",")[0]

    if tag == "-" {
        return
    }

    renderZeroVal := strings.Split(tag, ",")[1]

    // extract the value of the field
    fieldValue := reflect.ValueOf(n.High).Elem().FieldByName(key)
    f := fieldValue.Interface()
    value := reflect.ValueOf(f)

    if renderZeroVal != renderZero && (f == nil || value.IsZero()) {
        return
    }

    // create a new node entry
    nodeEntry := &NodeEntry{Tag: tagName, Key: key}
    if renderZeroVal == renderZero {
        nodeEntry.RenderZero = true
    }

    switch value.Kind() {
    case reflect.Float64, reflect.Float32:
        nodeEntry.Value = value.Float()
        x := float64(int(value.Float()*100)) / 100 // trim this down
        nodeEntry.StringValue = strconv.FormatFloat(x, 'f', -1, 64)
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        nodeEntry.Value = value.Int()
        nodeEntry.StringValue = value.String()
    case reflect.String:
        nodeEntry.Value = value.String()
    case reflect.Bool:
        nodeEntry.Value = value.Bool()
    case reflect.Slice:
        if tagName == "type" {
            if value.Len() == 1 {
                nodeEntry.Value = value.Index(0).String()
            } else {
                nodeEntry.Value = f
            }
        } else {
            if (renderZeroVal == renderZero) || (!value.IsNil() && !value.IsZero()) {
                nodeEntry.Value = f
            }
        }
    case reflect.Ptr:
        if !value.IsNil() {
            nodeEntry.Value = f
        }
    case reflect.Map:
        if !value.IsNil() && value.Len() > 0 {
            nodeEntry.Value = f
        }
    default:
        nodeEntry.Value = f
    }

    // if there is no low level object, then we cannot extract line numbers,
    // so skip and default to 0, which means a new entry to the spec.
    // this will place new content and the top of the rendered object.
    if n.Low != nil && !reflect.ValueOf(n.Low).IsZero() {
        lowFieldValue := reflect.ValueOf(n.Low).Elem().FieldByName(key)
        fLow := lowFieldValue.Interface()
        value = reflect.ValueOf(fLow)
        switch value.Kind() {

        case reflect.Slice:
            l := value.Len()
            lines := make([]int, l)
            for g := 0; g < l; g++ {
                qw := value.Index(g).Interface()
                if we, wok := qw.(low.HasKeyNode); wok {
                    lines[g] = we.GetKeyNode().Line
                }
            }
            sort.Ints(lines)
            nodeEntry.Line = lines[0] // pick the lowest line number so this key is sorted in order.
            break
        case reflect.Map:

            l := value.Len()
            lines := make([]int, l)
            for q, ky := range value.MapKeys() {
                if we, wok := ky.Interface().(low.HasKeyNode); wok {
                    lines[q] = we.GetKeyNode().Line
                }
            }
            sort.Ints(lines)
            nodeEntry.Line = lines[0] // pick the lowest line number, sort in order

        case reflect.Struct:
            y := value.Interface()
            nodeEntry.Line = 9999 + i
            if nb, ok := y.(low.HasValueNodeUntyped); ok {
                if nb.IsReference() {
                    if jk, kj := y.(low.HasKeyNode); kj {
                        nodeEntry.Line = jk.GetKeyNode().Line
                        break
                    }
                }
                if nb.GetValueNode() != nil {
                    nodeEntry.Line = nb.GetValueNode().Line
                }
            }
        default:
            // everything else, weight it to the bottom of the rendered object.
            // this is things that we have no way of knowing where they should be placed.
            nodeEntry.Line = 9999 + i
        }
    }
    if nodeEntry.Value != nil {
        n.Nodes = append(n.Nodes, nodeEntry)
    }
}

func (n *NodeBuilder) renderReference() []*yaml.Node {
    fg := n.Low.(low.IsReferenced)
    nodes := make([]*yaml.Node, 2)
    nodes[0] = utils.CreateStringNode("$ref")
    nodes[1] = utils.CreateStringNode(fg.GetReference())
    return nodes
}

// Render will render the NodeBuilder back to a YAML node, iterating over every NodeEntry defined
func (n *NodeBuilder) Render() *yaml.Node {
    if len(n.Nodes) == 0 {
        return utils.CreateEmptyMapNode()
    }

    // order nodes by line number, retain original order
    m := utils.CreateEmptyMapNode()
    if fg, ok := n.Low.(low.IsReferenced); ok {
        g := reflect.ValueOf(fg)
        if !g.IsNil() {
            if fg.IsReference() && !n.Resolve {
                m.Content = append(m.Content, n.renderReference()...)
                return m
            }
        }
    }

    sort.Slice(n.Nodes, func(i, j int) bool {
        if n.Nodes[i].Line != n.Nodes[j].Line {
            return n.Nodes[i].Line < n.Nodes[j].Line
        }
        return false
    })

    for i := range n.Nodes {
        node := n.Nodes[i]
        n.AddYAMLNode(m, node)
    }
    return m
}

// AddYAMLNode will add a new *yaml.Node to the parent node, using the tag, key and value provided.
// If the value is nil, then the node will not be added. This method is recursive, so it will dig down
// into any non-scalar types.
func (n *NodeBuilder) AddYAMLNode(parent *yaml.Node, entry *NodeEntry) *yaml.Node {
    if entry.Value == nil {
        return parent
    }

    // check the type
    t := reflect.TypeOf(entry.Value)
    var l *yaml.Node
    if entry.Tag != "" {
        l = utils.CreateStringNode(entry.Tag)
    }

    value := entry.Value
    line := entry.Line
    key := entry.Key

    var valueNode *yaml.Node
    switch t.Kind() {

    case reflect.String:
        val := value.(string)
        valueNode = utils.CreateStringNode(val)
        valueNode.Line = line
        break

    case reflect.Bool:
        val := value.(bool)
        if !val {
            valueNode = utils.CreateBoolNode("false")
        } else {
            valueNode = utils.CreateBoolNode("true")
        }
        valueNode.Line = line
        break

    case reflect.Int:
        val := strconv.Itoa(value.(int))
        valueNode = utils.CreateIntNode(val)
        valueNode.Line = line
        break

    case reflect.Int64:
        val := strconv.FormatInt(value.(int64), 10)
        valueNode = utils.CreateIntNode(val)
        valueNode.Line = line
        break

    case reflect.Float32:
        val := strconv.FormatFloat(float64(value.(float32)), 'f', 2, 64)
        valueNode = utils.CreateFloatNode(val)
        valueNode.Line = line
        break

    case reflect.Float64:
        precision := -1
        if entry.StringValue != "" && strings.Contains(entry.StringValue, ".") {
            precision = len(strings.Split(fmt.Sprint(entry.StringValue), ".")[1])
        }
        val := strconv.FormatFloat(value.(float64), 'f', precision, 64)
        valueNode = utils.CreateFloatNode(val)
        valueNode.Line = line
        break

    case reflect.Map:

        // the keys will be rendered randomly, if we don't find out the original line
        // number of the tag.

        var orderedCollection []*NodeEntry
        m := reflect.ValueOf(value)
        for g, k := range m.MapKeys() {
            var x string
            // extract key
            yu := k.Interface()
            if o, ok := yu.(low.HasKeyNode); ok {
                x = o.GetKeyNode().Value
            } else {
                x = k.String()
            }

            // go low and pull out the line number.
            lowProps := reflect.ValueOf(n.Low)
            if n.Low != nil && !lowProps.IsZero() && !lowProps.IsNil() {
                gu := lowProps.Elem()
                gi := gu.FieldByName(key)
                jl := reflect.ValueOf(gi)
                if !jl.IsZero() && gi.Interface() != nil {
                    gh := gi.Interface()
                    // extract low level key line number
                    if pr, ok := gh.(low.HasValueUnTyped); ok {
                        fg := reflect.ValueOf(pr.GetValueUntyped())
                        found := false
                        found, orderedCollection = n.extractLowMapKeys(fg, x, found, orderedCollection, m, k)
                        if found != true {
                            // this is something new, add it.
                            orderedCollection = append(orderedCollection, &NodeEntry{
                                Tag:   x,
                                Key:   x,
                                Line:  9999 + g,
                                Value: m.MapIndex(k).Interface(),
                            })
                        }
                    } else {
                        // this is a map, but it may be wrapped still.
                        bj := reflect.ValueOf(gh)
                        orderedCollection = n.extractLowMapKeysWrapped(bj, x, orderedCollection, g)
                    }
                } else {
                    // this is a map, without any low level details available (probably an extension map).
                    orderedCollection = append(orderedCollection, &NodeEntry{
                        Tag:   x,
                        Key:   x,
                        Line:  9999 + g,
                        Value: m.MapIndex(k).Interface(),
                    })
                }
            } else {
                // this is a map, without any low level details available (probably an extension map).
                orderedCollection = append(orderedCollection, &NodeEntry{
                    Tag:   x,
                    Key:   x,
                    Line:  9999 + g,
                    Value: m.MapIndex(k).Interface(),
                })
            }
        }

        // sort the slice by line number to ensure everything is rendered in order.
        sort.Slice(orderedCollection, func(i, j int) bool {
            return orderedCollection[i].Line < orderedCollection[j].Line
        })

        // create an empty map.
        p := utils.CreateEmptyMapNode()

        // build out each map node in original order.
        for _, cv := range orderedCollection {
            n.AddYAMLNode(p, cv)
        }
        if len(p.Content) > 0 {
            valueNode = p
        }

    case reflect.Slice:

        var rawNode yaml.Node
        m := reflect.ValueOf(value)
        sl := utils.CreateEmptySequenceNode()
        skip := false
        for i := 0; i < m.Len(); i++ {
            sqi := m.Index(i).Interface()
            // check if this is a reference.
            if glu, ok := sqi.(GoesLowUntyped); ok {
                if glu != nil {
                    ut := glu.GoLowUntyped()
                    if !reflect.ValueOf(ut).IsNil() {
                        r := ut.(low.IsReferenced)
                        if ut != nil && r.GetReference() != "" &&
                            ut.(low.IsReferenced).IsReference() {
                            if !n.Resolve {
                                refNode := utils.CreateRefNode(glu.GoLowUntyped().(low.IsReferenced).GetReference())
                                sl.Content = append(sl.Content, refNode)
                                skip = true
                            } else {
                                skip = false
                            }
                        } else {
                            skip = false
                        }
                    }
                }
            }
            if !skip {
                if er, ko := sqi.(Renderable); ko {
                    var rend interface{}
                    if !n.Resolve {
                        rend, _ = er.(Renderable).MarshalYAML()
                    } else {
                        // try and render inline, if we can, otherwise treat as normal.
                        if _, ko := er.(RenderableInline); ko {
                            rend, _ = er.(RenderableInline).MarshalYAMLInline()
                        } else {
                            rend, _ = er.(Renderable).MarshalYAML()
                        }
                    }
                    // check if this is a pointer or not.
                    if _, ok := rend.(*yaml.Node); ok {
                        sl.Content = append(sl.Content, rend.(*yaml.Node))
                    }
                    if _, ok := rend.(yaml.Node); ok {
                        k := rend.(yaml.Node)
                        sl.Content = append(sl.Content, &k)
                    }
                }
            }
        }

        if len(sl.Content) > 0 {
            valueNode = sl
            break
        }
        if skip {
            break
        }

        err := rawNode.Encode(value)
        if err != nil {
            return parent
        } else {
            valueNode = &rawNode
        }

    case reflect.Struct:
        if r, ok := value.(low.ValueReference[any]); ok {
            valueNode = r.GetValueNode()
            break
        }
        if r, ok := value.(low.ValueReference[string]); ok {
            valueNode = r.GetValueNode()
            break
        }
        if r, ok := value.(low.NodeReference[string]); ok {
            valueNode = r.GetValueNode()
            break
        }
        return parent

    case reflect.Ptr:
        if r, ok := value.(Renderable); ok {
            if gl, lg := value.(GoesLowUntyped); lg {
                if gl.GoLowUntyped() != nil {
                    ut := reflect.ValueOf(gl.GoLowUntyped())
                    if !ut.IsNil() {
                        if gl.GoLowUntyped().(low.IsReferenced).IsReference() {
                            if !n.Resolve {
                                // TODO: use renderReference here.
                                rvn := utils.CreateEmptyMapNode()
                                rvn.Content = append(rvn.Content, utils.CreateStringNode("$ref"))
                                rvn.Content = append(rvn.Content, utils.CreateStringNode(gl.GoLowUntyped().(low.IsReferenced).GetReference()))
                                valueNode = rvn
                                break
                            }
                        }
                    }
                }
            }
            var rawRender interface{}
            if !n.Resolve {
                rawRender, _ = r.MarshalYAML()
            } else {
                // try an inline render if we can, otherwise there is no option but to default to the
                // full render.
                if _, ko := r.(RenderableInline); ko {
                    rawRender, _ = r.(RenderableInline).MarshalYAMLInline()
                } else {
                    rawRender, _ = r.MarshalYAML()
                }
            }
            if rawRender != nil {
                if _, ko := rawRender.(*yaml.Node); ko {
                    valueNode = rawRender.(*yaml.Node)
                }
                if _, ko := rawRender.(yaml.Node); ko {
                    d := rawRender.(yaml.Node)
                    valueNode = &d
                }
            }
        } else {

            encodeSkip := false
            // check if the value is a bool, int or float
            if b, bok := value.(*bool); bok {
                encodeSkip = true
                if *b {
                    valueNode = utils.CreateBoolNode("true")
                    valueNode.Line = line
                } else {
                    if entry.RenderZero {
                        valueNode = utils.CreateBoolNode("false")
                        valueNode.Line = line
                    }
                }
            }
            if b, bok := value.(*int64); bok {
                encodeSkip = true
                if *b > 0 {
                    valueNode = utils.CreateIntNode(strconv.Itoa(int(*b)))
                    valueNode.Line = line
                }
            }
            if b, bok := value.(*float64); bok {
                encodeSkip = true
                if *b > 0 {
                    valueNode = utils.CreateFloatNode(strconv.FormatFloat(*b, 'f', -1, 64))
                    valueNode.Line = line
                }
            }
            if !encodeSkip {
                var rawNode yaml.Node
                err := rawNode.Encode(value)
                if err != nil {
                    return parent
                } else {
                    valueNode = &rawNode
                    valueNode.Line = line
                }
            }
        }

    }
    if valueNode == nil {
        return parent
    }
    if l != nil {
        parent.Content = append(parent.Content, l, valueNode)
    } else {
        parent.Content = valueNode.Content
    }
    return parent
}

func (n *NodeBuilder) extractLowMapKeysWrapped(iu reflect.Value, x string, orderedCollection []*NodeEntry, g int) []*NodeEntry {
    for _, ky := range iu.MapKeys() {
        ty := ky.Interface()
        if ere, eok := ty.(low.HasKeyNode); eok {
            er := ere.GetKeyNode().Value
            if er == x {
                orderedCollection = append(orderedCollection, &NodeEntry{
                    Tag:   x,
                    Key:   x,
                    Line:  ky.Interface().(low.HasKeyNode).GetKeyNode().Line,
                    Value: iu.MapIndex(ky).Interface(),
                })
            }
        } else {
            orderedCollection = append(orderedCollection, &NodeEntry{
                Tag:   x,
                Key:   x,
                Line:  9999 + g,
                Value: iu.MapIndex(ky).Interface(),
            })
        }
    }
    return orderedCollection
}

func (n *NodeBuilder) extractLowMapKeys(fg reflect.Value, x string, found bool, orderedCollection []*NodeEntry, m reflect.Value, k reflect.Value) (bool, []*NodeEntry) {
    for j, ky := range fg.MapKeys() {
        hu := ky.Interface()
        if we, wok := hu.(low.HasKeyNode); wok {
            er := we.GetKeyNode().Value
            if er == x {
                found = true
                orderedCollection = append(orderedCollection, &NodeEntry{
                    Tag:   x,
                    Key:   x,
                    Line:  we.GetKeyNode().Line,
                    Value: m.MapIndex(k).Interface(),
                })
            }
        } else {
            uu := ky.Interface()
            if uu == x {
                // this is a map, without any low level details available
                found = true
                orderedCollection = append(orderedCollection, &NodeEntry{
                    Tag:   uu.(string),
                    Key:   uu.(string),
                    Line:  9999 + j,
                    Value: m.MapIndex(k).Interface(),
                })
            }
        }
    }
    return found, orderedCollection
}

// Renderable is an interface that can be implemented by types that provide a custom MarshaYAML method.
type Renderable interface {
    MarshalYAML() (interface{}, error)
}

// RenderableInline is an interface that can be implemented by types that provide a custom MarshaYAML method.
type RenderableInline interface {
    MarshalYAMLInline() (interface{}, error)
}

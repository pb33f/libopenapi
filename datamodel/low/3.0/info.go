package v3

import (
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sync"
)

const (
	InfoLabel = "info"
)

type Info struct {
	Title          low.NodeReference[string]
	Description    low.NodeReference[string]
	TermsOfService low.NodeReference[string]
	Contact        low.NodeReference[*Contact]
	License        low.NodeReference[*License]
	Version        low.NodeReference[string]
}

func (i *Info) Build(root *yaml.Node) error {
	var wg sync.WaitGroup
	wg.Add(2)
	var errs []error

	contact := Contact{}
	_, kln, cn := utils.FindKeyNodeFull("contact", root.Content)
	go datamodel.BuildModelAsync(cn, &contact, &wg, &errs)

	license := License{}
	_, kln, ln := utils.FindKeyNodeFull("license", root.Content)
	go datamodel.BuildModelAsync(ln, &license, &wg, &errs)
	wg.Wait()
	i.Contact = low.NodeReference[*Contact]{Value: &contact, ValueNode: cn, KeyNode: kln}
	i.License = low.NodeReference[*License]{Value: &license, ValueNode: ln, KeyNode: kln}
	return nil
}

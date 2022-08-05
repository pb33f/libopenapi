package v3

import (
	"github.com/pb33f/libopenapi/datamodel/low"
)

type Discriminator struct {
	PropertyName low.NodeReference[string]
	Mapping      map[low.NodeReference[string]]low.NodeReference[string]
}

func (d *Discriminator) FindMappingValue(key string) *low.NodeReference[string] {
	for k, v := range d.Mapping {
		if k.Value == key {
			return &v
		}
	}
	return nil
}

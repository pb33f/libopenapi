// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package model

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChange_MarshalJSON(t *testing.T) {

	rinseAndRepeat := func(ch *Change) map[string]any {
		b, err := ch.MarshalJSON()
		assert.NoError(t, err)

		var rebuilt map[string]any
		err = json.Unmarshal(b, &rebuilt)
		assert.NoError(t, err)
		return rebuilt
	}

	change := Change{
		ChangeType: Modified,
	}
	rebuilt := rinseAndRepeat(&change)
	assert.Equal(t, "modified", rebuilt["changeText"])
	assert.Equal(t, float64(1), rebuilt["change"])

	change = Change{
		ChangeType: ObjectAdded,
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "object_added", rebuilt["changeText"])
	assert.Equal(t, float64(3), rebuilt["change"])

	change = Change{
		ChangeType: ObjectRemoved,
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "object_removed", rebuilt["changeText"])
	assert.Equal(t, float64(4), rebuilt["change"])

	change = Change{
		ChangeType: PropertyAdded,
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "property_added", rebuilt["changeText"])
	assert.Equal(t, float64(2), rebuilt["change"])

	change = Change{
		ChangeType: PropertyRemoved,
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "property_removed", rebuilt["changeText"])
	assert.Equal(t, float64(5), rebuilt["change"])

	change = Change{
		Original: "gangster",
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "gangster", rebuilt["original"])

	change = Change{
		New: "shoes",
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "shoes", rebuilt["new"])

	one := 1
	change = Change{
		Context: &ChangeContext{
			OriginalLine: &one,
		},
	}
	rebuilt = rinseAndRepeat(&change)
	assert.NotNil(t, rebuilt["context"])

	change = Change{
		Type: "burger",
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "burger", rebuilt["type"])

	change = Change{
		Path: "difficult",
	}
	rebuilt = rinseAndRepeat(&change)
	assert.Equal(t, "difficult", rebuilt["path"])

	prop := &PropertyChanges{Changes: []*Change{&change}}
	assert.Empty(t, prop.GetPropertyChanges())

}

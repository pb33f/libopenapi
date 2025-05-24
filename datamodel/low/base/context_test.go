// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package base

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetModelContext(t *testing.T) {
	assert.Nil(t, GetModelContext(nil))
	assert.Nil(t, GetModelContext(context.Background()))

	ctx := context.WithValue(context.Background(), "modelCtx", &ModelContext{})
	assert.NotNil(t, GetModelContext(ctx))

	ctx = context.WithValue(context.Background(), "modelCtx", "wrong")
	assert.Nil(t, GetModelContext(ctx))
}

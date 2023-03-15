// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
    "github.com/stretchr/testify/assert"
    "strings"
    "testing"
)

func TestDynamicValue_Render_A(t *testing.T) {
    dv := &DynamicValue[string, int]{N: 0, A: "hello"}
    dvb, _ := dv.Render()
    assert.Equal(t, "hello", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_B(t *testing.T) {
    dv := &DynamicValue[string, int]{N: 1, B: 12345}
    dvb, _ := dv.Render()
    assert.Equal(t, "12345", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Bool(t *testing.T) {
    dv := &DynamicValue[string, bool]{N: 1, B: true}
    dvb, _ := dv.Render()
    assert.Equal(t, "true", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Int64(t *testing.T) {
    dv := &DynamicValue[string, int64]{N: 1, B: 12345567810}
    dvb, _ := dv.Render()
    assert.Equal(t, "12345567810", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Int32(t *testing.T) {
    dv := &DynamicValue[string, int32]{N: 1, B: 1234567891}
    dvb, _ := dv.Render()
    assert.Equal(t, "1234567891", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Float32(t *testing.T) {
    dv := &DynamicValue[string, float32]{N: 1, B: 23456.123}
    dvb, _ := dv.Render()
    assert.Equal(t, "23456.123", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Float64(t *testing.T) {
    dv := &DynamicValue[string, float64]{N: 1, B: 23456.1233456778}
    dvb, _ := dv.Render()
    assert.Equal(t, "23456.1233456778", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_Ptr(t *testing.T) {

    type cake struct {
        Cake string
    }

    dv := &DynamicValue[string, *cake]{N: 1, B: &cake{Cake: "vanilla"}}
    dvb, _ := dv.Render()
    assert.Equal(t, "cake: vanilla", strings.TrimSpace(string(dvb)))
}

func TestDynamicValue_Render_PtrRenderable(t *testing.T) {

    tag := &Tag{
        Name: "cake",
    }

    dv := &DynamicValue[string, *Tag]{N: 1, B: tag}
    dvb, _ := dv.Render()
    assert.Equal(t, "name: cake", strings.TrimSpace(string(dvb)))
}

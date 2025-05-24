// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAreValuesCorrectlyTyped(t *testing.T) {
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{"hi"}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{1}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{"nice", 123, int64(12345)}), 2)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{1.2, "burgers"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{true, false, "what"}), 2)

	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{1, 2, 3, 4}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{"no way!"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{"nice", 123, int64(12345)}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{999, 1.2, "burgers"}), 2)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{true, false, "what"}), 3)

	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{1.2345}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{"no way!"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{"nice", 123, 2.353}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{999, 1.2, "burgers"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{true, false, "what"}), 3)

	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{true, false, true}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{"no way!"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{"nice", 123, 2.353, true}), 3)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{true, true, "burgers"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{true, false, "what", 1.2, 4}), 3)
	assert.Nil(t, AreValuesCorrectlyTyped("boolean", []string{"hi"}))
}

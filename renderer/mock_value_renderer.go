// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package renderer

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

func (wr *SchemaRenderer) renderMockStringValue(schema *base.Schema, key string, maxGeneratedStringBytes int) any {
	if schema == nil {
		return nil
	}
	if schema.Enum != nil && len(schema.Enum) > 0 {
		enum := schema.Enum[wr.rand.Int()%len(schema.Enum)]
		var example any
		_ = enum.Decode(&example)
		return example
	}

	var minLength int64 = 3
	var maxLength int64 = 10
	if schema.MinLength != nil {
		minLength = *schema.MinLength
	}
	hasSchemaMaxLength := schema.MaxLength != nil
	schemaMaxLength := maxLength
	if schema.MaxLength != nil {
		maxLength = *schema.MaxLength
		schemaMaxLength = *schema.MaxLength
	}
	minLength, maxLength = boundedGeneratedStringRange(minLength, maxLength, maxGeneratedStringBytes)
	limitGeneratedString := func(value string) string {
		return truncateStringBytes(value, maxGeneratedStringBytes)
	}
	randomWord := func() string {
		return limitGeneratedString(wr.RandomWord(minLength, maxLength, 0))
	}

	if schema.Examples != nil && len(schema.Examples) > 0 {
		if len(schema.Examples) > 1 && key == itemsType {
			renderedExamples := make([]any, len(schema.Examples))
			for i, exmp := range schema.Examples {
				if exmp != nil {
					var ex any
					_ = exmp.Decode(&ex)
					renderedExamples[i] = fmt.Sprint(ex)
				}
			}
			return renderedExamples
		}
		var renderedExample any
		if exmp := schema.Examples[0]; exmp != nil {
			var ex any
			_ = exmp.Decode(&ex)
			renderedExample = fmt.Sprint(ex)
		}
		return renderedExample
	}

	switch schema.Format {
	case dateTimeType:
		return limitGeneratedString(time.Now().Format(time.RFC3339))
	case dateType:
		return limitGeneratedString(time.Now().Format("2006-01-02"))
	case timeType:
		return limitGeneratedString(time.Now().Format("15:04:05"))
	case emailType:
		return limitGeneratedString(fmt.Sprintf("%s@%s.com", randomWord(), randomWord()))
	case hostnameType:
		return limitGeneratedString(fmt.Sprintf("%s.com", randomWord()))
	case ipv4Type:
		return limitGeneratedString(fmt.Sprintf("%d.%d.%d.%d",
			wr.rand.Int()%255, wr.rand.Int()%255, wr.rand.Int()%255, wr.rand.Int()%255))
	case ipv6Type:
		return limitGeneratedString(fmt.Sprintf("%04x:%04x:%04x:%04x:%04x:%04x:%04x:%04x",
			wr.rand.Intn(65535), wr.rand.Intn(65535), wr.rand.Intn(65535), wr.rand.Intn(65535),
			wr.rand.Intn(65535), wr.rand.Intn(65535), wr.rand.Intn(65535), wr.rand.Intn(65535),
		))
	case uriType:
		return limitGeneratedString(fmt.Sprintf("https://%s-%s-%s.com/%s",
			randomWord(), randomWord(), randomWord(), randomWord()))
	case uriReferenceType:
		return limitGeneratedString(fmt.Sprintf("/%s/%s", randomWord(), randomWord()))
	case uuidType:
		return limitGeneratedString(wr.PseudoUUID())
	case byteType, passwordType:
		return randomWord()
	case binaryType:
		return limitGeneratedString(base64.StdEncoding.EncodeToString([]byte(randomWord())))
	case bigIntType:
		return limitGeneratedString(fmt.Sprint(wr.RandomInt(minLength, maxLength)))
	case decimalType:
		return limitGeneratedString(fmt.Sprint(wr.RandomFloat64()))
	default:
		if schema.Pattern != "" {
			str, err := wr.generatePatternString(schema.Pattern, schemaMaxLength, hasSchemaMaxLength)
			if err == nil {
				return str
			}
		}
		return randomWord()
	}
}

func (wr *SchemaRenderer) renderMockNumberValue(schema *base.Schema) any {
	if schema == nil {
		return nil
	}
	if schema.Enum != nil && len(schema.Enum) > 0 {
		enum := schema.Enum[wr.rand.Int()%len(schema.Enum)]
		var example any
		_ = enum.Decode(&example)
		return example
	}

	var minimum int64 = 1
	var maximum int64 = 100
	if schema.Minimum != nil {
		minimum = int64(*schema.Minimum)
	}
	if schema.Maximum != nil {
		maximum = int64(*schema.Maximum)
	}

	if schema.Examples != nil && len(schema.Examples) > 0 {
		var renderedExample any
		if exmp := schema.Examples[0]; exmp != nil {
			var ex any
			_ = exmp.Decode(&ex)
			renderedExample = ex
		}
		return renderedExample
	}

	switch schema.Format {
	case floatType:
		return wr.rand.Float32()
	case doubleType:
		return wr.rand.Float64()
	case int32Type:
		return int(wr.RandomInt(minimum, maximum))
	case bigIntType:
		return wr.RandomInt(minimum, maximum)
	case decimalType:
		return wr.RandomFloat64()
	default:
		return wr.RandomInt(minimum, maximum)
	}
}

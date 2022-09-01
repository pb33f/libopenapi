// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

//
//import (
//    v2 "github.com/pb33f/libopenapi/datamodel/high/2.0"
//    v3 "github.com/pb33f/libopenapi/datamodel/high/3.0"
//)
//
//type Document struct {
//	version string
//	info    *SpecInfo
//}
//
//func (d *Document) GetVersion() string {
//	return d.version
//}
//
//func (d *Document) BuildV2Document() (*v2.Swagger, error) {
//	return nil, nil
//}
//
//func (d *Document) BuildV3Document() (*v3.Document, error) {
//	return nil, nil
//}
//
//func LoadDocument(specBytes []byte) (*Document, error) {
//	info, err := ExtractSpecInfo(specBytes)
//	if err != nil {
//		return nil, err
//	}
//	return &Document{info: info, version: info.Version}, nil
//}

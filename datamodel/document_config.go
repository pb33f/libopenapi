// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package datamodel

import "net/url"

// DocumentConfiguration is used to configure the document creation process. It was added in v0.6.0 to allow
// for more fine-grained control over controls and new features.
//
// The default configuration will set AllowFileReferences to false and AllowRemoteReferences to false, which means
// any non-local (local being the specification, not the file system) references, will be ignored.
type DocumentConfiguration struct {
    // if the document uses relative file references, this is the base url to use when resolving them.
    BaseURL *url.URL

    // AllowFileReferences will allow the index to locate relative file references. This is disabled by default.
    AllowFileReferences bool

    // AllowRemoteReferences will allow the index to lookup remote references. This is disabled by default.
    AllowRemoteReferences bool
}

// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
    "fmt"
    "github.com/pb33f/libopenapi/utils"
    "github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
    "gopkg.in/yaml.v3"
    "io/ioutil"
    "strings"
)

// FindComponent will locate a component by its reference, returns nil if nothing is found.
// This method will recurse through remote, local and file references. For each new external reference
// a new index will be created. These indexes can then be traversed recursively.
func (index *SpecIndex) FindComponent(componentId string, parent *yaml.Node) *Reference {
    if index.root == nil {
        return nil
    }

    remoteLookup := func(id string) (*yaml.Node, *yaml.Node, error) {
        if index.config.AllowRemoteLookup {
            return index.lookupRemoteReference(id)
        } else {
            return nil, nil, fmt.Errorf("remote lookups are not premitted, " +
                "please set AllowRemoteLookup to true in the configuration")
        }
    }

    fileLookup := func(id string) (*yaml.Node, *yaml.Node, error) {
        if index.config.AllowFileLookup {
            return index.lookupFileReference(id)
        } else {
            return nil, nil, fmt.Errorf("local lookups are not permitted, " +
                "please set AllowFileLookup to true in the configuration")
        }
    }

    switch DetermineReferenceResolveType(componentId) {
    case LocalResolve: // ideally, every single ref in every single spec is local. however, this is not the case.
        return index.FindComponentInRoot(componentId)

    case HttpResolve:
        uri := strings.Split(componentId, "#")
        if len(uri) >= 2 {
            return index.performExternalLookup(uri, componentId, remoteLookup, parent)
        }
        if len(uri) == 1 {
            // if there is no reference, second segment is empty / has no name
            // this means there is no component to look-up and the entire file should be pulled in.
            // to stop all the other code from breaking (that is expecting a component), let's just post-pend
            // a hash to the end of the componentId and ensure the uri slice is as expected.
            // described in https://github.com/pb33f/libopenapi/issues/37
            componentId = fmt.Sprintf("%s#", componentId)
            uri = append(uri, "")
            return index.performExternalLookup(uri, componentId, remoteLookup, parent)
        }

    case FileResolve:
        uri := strings.Split(componentId, "#")
        if len(uri) == 2 {
            return index.performExternalLookup(uri, componentId, fileLookup, parent)
        }
        if len(uri) == 1 {
            // if there is no reference, second segment is empty / has no name
            // this means there is no component to look-up and the entire file should be pulled in.
            // to stop all the other code from breaking (that is expecting a component), let's just post-pend
            // a hash to the end of the componentId and ensure the uri slice is as expected.
            // described in https://github.com/pb33f/libopenapi/issues/37
            //
            // ^^ this same issue was re-reported in file based lookups in vacuum.
            // more info here: https://github.com/daveshanley/vacuum/issues/225
            componentId = fmt.Sprintf("%s#", componentId)
            uri = append(uri, "")
            return index.performExternalLookup(uri, componentId, fileLookup, parent)
        }
    }
    return nil
}

func (index *SpecIndex) lookupRemoteReference(ref string) (*yaml.Node, *yaml.Node, error) {
    // split string to remove file reference
    uri := strings.Split(ref, "#")

    var parsedRemoteDocument *yaml.Node
    if index.seenRemoteSources[uri[0]] != nil {
        parsedRemoteDocument = index.seenRemoteSources[uri[0]]
    } else {
        index.httpLock.Lock()
        resp, err := index.httpClient.Get(uri[0])
        index.httpLock.Unlock()
        if err != nil {
            return nil, nil, err
        }
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            return nil, nil, err
        }

        var remoteDoc yaml.Node
        err = yaml.Unmarshal(body, &remoteDoc)
        if err != nil {
            return nil, nil, err
        }
        parsedRemoteDocument = &remoteDoc
        index.remoteLock.Lock()
        index.seenRemoteSources[uri[0]] = &remoteDoc
        index.remoteLock.Unlock()
    }

    // lookup item from reference by using a path query.
    var query string
    if len(uri) >= 2 {
        query = fmt.Sprintf("$%s", strings.ReplaceAll(uri[1], "/", "."))
    } else {
        query = "$"
    }

    // remove any URL encoding
    query = strings.Replace(query, "~1", "./", 1)
    query = strings.ReplaceAll(query, "~1", "/")

    path, err := yamlpath.NewPath(query)
    if err != nil {
        return nil, nil, err
    }
    result, _ := path.Find(parsedRemoteDocument)
    if len(result) == 1 {
        return result[0], parsedRemoteDocument, nil
    }
    return nil, nil, nil
}

func (index *SpecIndex) lookupFileReference(ref string) (*yaml.Node, *yaml.Node, error) {
    // split string to remove file reference
    uri := strings.Split(ref, "#")

    file := strings.ReplaceAll(uri[0], "file:", "")

    var parsedRemoteDocument *yaml.Node

    if index.seenRemoteSources[file] != nil {
        parsedRemoteDocument = index.seenRemoteSources[file]

    } else {

        // try and read the file off the local file system, if it fails
        // check for a baseURL and then ask our remote lookup function to go try and get it.
        // index.fileLock.Lock()
        body, err := ioutil.ReadFile(file)
        // index.fileLock.Unlock()

        if err != nil {

            // if we have a baseURL, then we can try and get the file from there.
            if index.config != nil && index.config.BaseURL != nil {

                u := index.config.BaseURL
                remoteRef := fmt.Sprintf("%s://%s%s/%s", u.Scheme, u.Host, u.Path, ref)
                a, b, e := index.lookupRemoteReference(remoteRef)
                if e != nil {
                    // give up, we can't find the file, not locally, not remotely. It's toast.
                    return nil, nil, e
                }

                // everything looks good, lets just make sure we also add a key to the raw reference name.
                if _, ok := index.seenRemoteSources[file]; !ok {
                    index.seenRemoteSources[file] = b
                }

                return a, b, nil

            } else {
                // no baseURL? then we can't do anything, give up.
                return nil, nil, err
            }
        }

        var remoteDoc yaml.Node
        err = yaml.Unmarshal(body, &remoteDoc)
        if err != nil {
            return nil, nil, err
        }
        parsedRemoteDocument = &remoteDoc
        index.seenRemoteSources[file] = &remoteDoc
    }

    // lookup item from reference by using a path query.
    var query string
    if len(uri) >= 2 {
        query = fmt.Sprintf("$%s", strings.ReplaceAll(uri[1], "/", "."))
    } else {
        query = "$"
    }

    // remove any URL encoding
    query = strings.Replace(query, "~1", "./", 1)
    query = strings.ReplaceAll(query, "~1", "/")

    path, err := yamlpath.NewPath(query)
    if err != nil {
        return nil, nil, err
    }
    result, _ := path.Find(parsedRemoteDocument)
    if len(result) == 1 {
        return result[0], parsedRemoteDocument, nil
    }

    return nil, parsedRemoteDocument, nil
}

func (index *SpecIndex) FindComponentInRoot(componentId string) *Reference {
    if index.root != nil {
        name, friendlySearch := utils.ConvertComponentIdIntoFriendlyPathSearch(componentId)
        path, err := yamlpath.NewPath(friendlySearch)
        if path == nil || err != nil {
            return nil // no component found
        }
        res, _ := path.Find(index.root)
        if len(res) == 1 {
            ref := &Reference{
                Definition:            componentId,
                Name:                  name,
                Node:                  res[0],
                Path:                  friendlySearch,
                RequiredRefProperties: index.extractDefinitionRequiredRefProperties(res[0], map[string][]string{}),
            }

            return ref
        }
    }
    return nil
}

func (index *SpecIndex) performExternalLookup(uri []string, componentId string,
    lookupFunction ExternalLookupFunction, parent *yaml.Node,
) *Reference {
    if len(uri) > 0 {
        externalSpecIndex := index.externalSpecIndex[uri[0]]
        if externalSpecIndex == nil {
            _, newRoot, err := lookupFunction(componentId)
            if err != nil {
                indexError := &IndexingError{
                    Err:  err,
                    Node: parent,
                    Path: componentId,
                }
                index.refErrors = append(index.refErrors, indexError)
                return nil
            }

            // cool, cool, lets index this spec also. This is a recursive action and will keep going
            // until all remote references have been found.
            newIndex := NewSpecIndexWithConfig(newRoot, index.config)
            index.fileLock.Lock()
            index.externalSpecIndex[uri[0]] = newIndex
            index.fileLock.Unlock()
            externalSpecIndex = newIndex
        }

        foundRef := externalSpecIndex.FindComponentInRoot(uri[1])
        if foundRef != nil {
            nameSegs := strings.Split(uri[1], "/")
            ref := &Reference{
                Definition:     componentId,
                Name:           nameSegs[len(nameSegs)-1],
                Node:           foundRef.Node,
                IsRemote:       true,
                RemoteLocation: componentId,
                Path:           foundRef.Path,
            }
            return ref
        }
    }
    return nil
}

// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"fmt"
	"github.com/pb33f/libopenapi/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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
			return nil, nil, fmt.Errorf("remote lookups are not permitted, " +
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

var httpClient = &http.Client{Timeout: time.Duration(60) * time.Second}

func getRemoteDoc(u string, d chan []byte, e chan error) {
	resp, err := httpClient.Get(u)
	if err != nil {
		e <- err
		close(e)
		close(d)
		return
	}
	var body []byte
	body, _ = ioutil.ReadAll(resp.Body)
	d <- body
	close(e)
	close(d)
}

func (index *SpecIndex) lookupRemoteReference(ref string) (*yaml.Node, *yaml.Node, error) {
	// split string to remove file reference
	uri := strings.Split(ref, "#")

	// have we already seen this remote source?
	var parsedRemoteDocument *yaml.Node
	alreadySeen, foundDocument := index.CheckForSeenRemoteSource(uri[0])

	if alreadySeen {
		parsedRemoteDocument = foundDocument
	} else {

		d := make(chan bool)
		var body []byte
		var err error

		go func(uri string) {
			bc := make(chan []byte)
			ec := make(chan error)
			go getRemoteDoc(uri, bc, ec)
			select {
			case v := <-bc:
				body = v
				break
			case er := <-ec:
				err = er
				break
			}
			var remoteDoc yaml.Node
			er := yaml.Unmarshal(body, &remoteDoc)
			if er != nil {
				err = er
				d <- true
				return
			}
			parsedRemoteDocument = &remoteDoc
			if index.config != nil {
				index.config.seenRemoteSources.Store(uri, &remoteDoc)
			}
			d <- true
		}(uri[0])

		// wait for double go fun.
		<-d
		if err != nil {
			// no bueno.
			return nil, nil, err
		}
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
	filePath := filepath.Dir(file)
	fileName := filepath.Base(file)

	var parsedRemoteDocument *yaml.Node

	if index.seenRemoteSources[file] != nil {
		parsedRemoteDocument = index.seenRemoteSources[file]
	} else {

		base := index.config.BasePath
		fileToRead := filepath.Join(base, filePath, fileName)

		// try and read the file off the local file system, if it fails
		// check for a baseURL and then ask our remote lookup function to go try and get it.
		body, err := os.ReadFile(fileToRead)

		if err != nil {

			// if we have a baseURL, then we can try and get the file from there.
			if index.config != nil && index.config.BaseURL != nil {

				u := index.config.BaseURL
				remoteRef := GenerateCleanSpecConfigBaseURL(u, ref, true)
				a, b, e := index.lookupRemoteReference(remoteRef)
				if e != nil {
					// give up, we can't find the file, not locally, not remotely. It's toast.
					return nil, nil, e
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
		if index.seenLocalSources != nil {
			index.sourceLock.Lock()
			index.seenLocalSources[file] = &remoteDoc
			index.sourceLock.Unlock()
		}
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

		// check component for url encoding.
		if strings.Contains(componentId, "%") {
			// decode the url.
			componentId, _ = url.QueryUnescape(componentId)
		}

		name, friendlySearch := utils.ConvertComponentIdIntoFriendlyPathSearch(componentId)
		path, err := yamlpath.NewPath(friendlySearch)
		if path == nil || err != nil {
			return nil // no component found
		}
		res, _ := path.Find(index.root)

		if len(res) == 1 {
			resNode := res[0]
			if res[0].Kind == yaml.DocumentNode {
				resNode = res[0].Content[0]
			}
			ref := &Reference{
				Definition:            componentId,
				Name:                  name,
				Node:                  resNode,
				Path:                  friendlySearch,
				RequiredRefProperties: index.extractDefinitionRequiredRefProperties(res[0], map[string][]string{}),
			}

			return ref
		}
	}
	return nil
}

func (index *SpecIndex) performExternalLookup(uri []string, componentId string,
	lookupFunction ExternalLookupFunction, parent *yaml.Node) *Reference {
	if len(uri) > 0 {
		index.externalLock.RLock()
		externalSpecIndex := index.externalSpecIndex[uri[0]]
		index.externalLock.RUnlock()

		if externalSpecIndex == nil {
			_, newRoot, err := lookupFunction(componentId)
			if err != nil {
				indexError := &IndexingError{
					Err:  err,
					Node: parent,
					Path: componentId,
				}
				index.errorLock.Lock()
				index.refErrors = append(index.refErrors, indexError)
				index.errorLock.Unlock()
				return nil
			}

			// cool, cool, lets index this spec also. This is a recursive action and will keep going
			// until all remote references have been found.
			var bp *url.URL
			var bd string

			if index.config.BaseURL != nil {
				bp = index.config.BaseURL
			}
			if index.config.BasePath != "" {
				bd = index.config.BasePath
			}

			var path, newBasePath string
			var newUrl *url.URL

			if bp != nil {
				path = GenerateCleanSpecConfigBaseURL(bp, uri[0], false)
				newUrl, _ = url.Parse(path)
				newBasePath = filepath.Dir(filepath.Join(index.config.BasePath, filepath.Dir(newUrl.Path)))
			}
			if bd != "" {
				if len(uri[0]) > 0 {
					// if there is no base url defined, but we can know we have been requested remotely,
					// set the base url to the remote url base path.
					// first check if the first param is actually a URL
					io, er := url.ParseRequestURI(uri[0])
					if er != nil {
						newBasePath = filepath.Dir(filepath.Join(bd, uri[0]))
					} else {
						if newUrl == nil || newUrl.String() != io.String() {
							newUrl, _ = url.Parse(fmt.Sprintf("%s://%s%s", io.Scheme, io.Host, filepath.Dir(io.Path)))
						}
						newBasePath = filepath.Dir(filepath.Join(bd, uri[1]))
					}
				} else {
					newBasePath = filepath.Dir(filepath.Join(bd, uri[0]))
				}
			}

			if newUrl != nil || newBasePath != "" {
				newConfig := &SpecIndexConfig{
					BaseURL:           newUrl,
					BasePath:          newBasePath,
					AllowRemoteLookup: index.config.AllowRemoteLookup,
					AllowFileLookup:   index.config.AllowFileLookup,
					ParentIndex:       index,
					seenRemoteSources: index.config.seenRemoteSources,
					remoteLock:        index.config.remoteLock,
					uri:               uri,
				}

				var newIndex *SpecIndex
				seen := index.SearchAncestryForSeenURI(uri[0])
				if seen == nil {

					newIndex = NewSpecIndexWithConfig(newRoot, newConfig)
					index.refLock.Lock()
					index.externalLock.Lock()
					index.externalSpecIndex[uri[0]] = newIndex
					index.externalLock.Unlock()
					newIndex.relativePath = path
					newIndex.parentIndex = index
					index.AddChild(newIndex)
					index.refLock.Unlock()
					externalSpecIndex = newIndex
				} else {
					externalSpecIndex = seen
				}
			}
		}

		if externalSpecIndex != nil {
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
	}
	return nil
}

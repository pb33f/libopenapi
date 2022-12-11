// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package reports

import (
    v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
    "github.com/pb33f/libopenapi/what-changed/model"
)

// Changed provides a simple wrapper for changed counts
type Changed struct {
    Total    int `json:"totalChanges"`
    Breaking int `json:"breakingChanges"`
}

// OverallReport provides a Document level overview of all changes to an OpenAPI doc.
type OverallReport struct {
    ChangeReport map[string]*Changed `json:"overallSummaryReport"`
}

// CreateOverallReport will create a high level report for all top level changes (but with deep counts)
func CreateOverallReport(changes *model.DocumentChanges) *OverallReport {

    changedReport := make(map[string]*Changed)
    if changes.InfoChanges != nil {
        changedReport[v3.InfoLabel] = createChangedModel(changes.InfoChanges)
    }
    if changes.PathsChanges != nil {
        changedReport[v3.PathsLabel] = createChangedModel(changes.PathsChanges)
    }
    if changes.TagChanges != nil {
        j := make([]HasChanges, len(changes.TagChanges))
        for k := range changes.TagChanges {
            j[k] = HasChanges(changes.TagChanges[k])
        }
        changedReport[v3.TagsLabel] = createChangedModelFromSlice(j)
    }
    if changes.ExternalDocChanges != nil {
        changedReport[v3.ExternalDocsLabel] = createChangedModel(changes.ExternalDocChanges)
    }
    if changes.WebhookChanges != nil {
        j := make([]HasChanges, len(changes.WebhookChanges))
        z := 0
        for k := range changes.WebhookChanges {
            j[z] = HasChanges(changes.WebhookChanges[k])
            z++
        }
        ch := createChangedModelFromSlice(j)
        if ch.Total > 0 {
            changedReport[v3.WebhooksLabel] = ch
        }
    }
    if changes.ServerChanges != nil {
        j := make([]HasChanges, len(changes.ServerChanges))
        for k := range changes.ServerChanges {
            j[k] = HasChanges(changes.ServerChanges[k])
        }
        changedReport[v3.ServersLabel] = createChangedModelFromSlice(j)
    }
    if changes.SecurityRequirementChanges != nil {
        j := make([]HasChanges, len(changes.SecurityRequirementChanges))
        for k := range changes.SecurityRequirementChanges {
            j[k] = HasChanges(changes.SecurityRequirementChanges[k])
        }
        changedReport[v3.SecurityLabel] = createChangedModelFromSlice(j)
    }
    if changes.ComponentsChanges != nil {
        changedReport[v3.ComponentsLabel] = createChangedModel(changes.ComponentsChanges)
    }
    return &OverallReport{
        ChangeReport: changedReport,
    }
}

func createChangedModel(ch HasChanges) *Changed {
    return &Changed{ch.TotalChanges(), ch.TotalBreakingChanges()}
}

func createChangedModelFromSlice(ch []HasChanges) *Changed {
    t := 0
    b := 0
    for n := range ch {
        t += ch[n].TotalChanges()
        b += ch[n].TotalBreakingChanges()
    }
    return &Changed{t, b}
}

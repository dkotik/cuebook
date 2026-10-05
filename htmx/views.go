package htmx

import (
	"fmt"
	"net/http"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
)

type fileLink struct {
	Name string
	URL  string
}

type pageData struct {
	Files         []fileLink
	Selected      string
	Entries       []entryView
	AddFields     []addFieldView
	ReadOnly      bool
	Error         string
	DocumentError string
}

type entryView struct {
	Index   int
	File    string
	Title   string
	Fields  []fieldView
	Details []fieldView
}

type fieldView struct {
	File      string
	Index     int
	Name      string
	Value     string
	MultiLine bool
	Secret    bool
	ReadOnly  bool
}

type addFieldView struct {
	Name      string
	Value     string
	MultiLine bool
	Secret    bool
	Optional  bool
}

type entryFieldDefinition struct {
	Field    cuebook.Field
	Optional bool
}

func (a *handler) pageForDocument(fileName string, fileNames []string, document cuebook.Document, notice string) (pageData, int) {
	data := a.basePage(fileNames, notice)
	data.Selected = fileName
	if !data.ReadOnly {
		data.AddFields = makeAddFieldViews(document)
	}
	entries, err := makeEntryViews(document, fileName, data.ReadOnly)
	if err != nil {
		data.DocumentError = "Unable to display this CUE document: " + err.Error()
		return data, http.StatusUnprocessableEntity
	}
	data.Entries = entries
	return data, http.StatusOK
}

func (a *handler) basePage(fileNames []string, notice string) pageData {
	data := pageData{
		ReadOnly: a.committer == nil,
		Error:    notice,
		Files:    make([]fileLink, 0, len(fileNames)),
	}
	for _, name := range fileNames {
		data.Files = append(data.Files, fileLink{Name: name, URL: fileURL(name)})
	}
	return data
}

func makeEntryViews(document cuebook.Document, fileName string, readOnly bool) ([]entryView, error) {
	var result []entryView
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", index, err)
		}
		view := entryView{
			Index: index,
			File:  fileName,
			Title: entry.GetTitle(),
		}
		if view.Title == "" {
			view.Title = fmt.Sprintf("Entry %d", index+1)
		}
		for _, field := range entry.Fields {
			view.Fields = append(view.Fields, makeFieldView(field, fileName, index, readOnly))
		}
		for _, field := range entry.Details {
			view.Details = append(view.Details, makeFieldView(field, fileName, index, readOnly))
		}
		result = append(result, view)
		index++
	}
	return result, nil
}

func makeAddFieldViews(document cuebook.Document) []addFieldView {
	var result []addFieldView
	for _, definition := range entryFieldDefinitions(document) {
		field := definition.Field
		_, secret := metadata.GetFieldAttributes(field.Value, "cuebook").GetFirstOf("argon2id")
		view := addFieldView{
			Name:      field.Name,
			MultiLine: metadata.IsMultiLine(field.Value),
			Secret:    secret,
			Optional:  definition.Optional,
		}
		if !secret {
			if value, ok := field.Default(); ok {
				view.Value = value
			} else if field.Value.IsConcrete() {
				view.Value = field.String()
			}
		}
		if field.Value.IncompleteKind()&cue.StringKind == 0 && field.Value.IncompleteKind()&(cue.ListKind|cue.StructKind) != 0 {
			view.MultiLine = true
		}
		result = append(result, view)
	}
	return result
}

func entryFieldDefinitions(document cuebook.Document) []entryFieldDefinition {
	var result []entryFieldDefinition
	for selector, value := range cuebook.EachFieldDefinition(document.Value) {
		result = append(result, entryFieldDefinition{
			Field:    cuebook.Field{Name: selector.Unquoted(), Value: value},
			Optional: selector.ConstraintType() == cue.OptionalConstraint,
		})
	}
	return result
}

func makeFieldView(field cuebook.Field, fileName string, index int, readOnly bool) fieldView {
	_, secret := metadata.GetFieldAttributes(field.Value, "cuebook").GetFirstOf("argon2id")
	return fieldView{
		File:      fileName,
		Index:     index,
		Name:      field.Name,
		Value:     field.String(),
		MultiLine: metadata.IsMultiLine(field.Value),
		Secret:    secret,
		ReadOnly:  readOnly,
	}
}

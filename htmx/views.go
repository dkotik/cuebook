package htmx

import (
	"fmt"
	"net/http"

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

func (a *handler) pageForDocument(fileName string, fileNames []string, document cuebook.Document, notice string) (pageData, int) {
	data := a.basePage(fileNames, notice)
	data.Selected = fileName
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

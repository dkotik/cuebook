package htmx

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

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

func (a *handler) index(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	data, status := a.loadPage(fileName, "")
	a.renderPage(w, r, data, status)
}

func (a *handler) loadPage(fileName, notice string) (pageData, int) {
	data := pageData{ReadOnly: a.committer == nil, Error: notice}
	fileNames, err := a.fileNames()
	if err != nil {
		data.Error = "Unable to list CUE files."
		return data, http.StatusInternalServerError
	}
	data.Files = make([]fileLink, 0, len(fileNames))
	for _, name := range fileNames {
		data.Files = append(data.Files, fileLink{
			Name: name,
			URL:  fileURL(name),
		})
	}

	if fileName == "" {
		return data, http.StatusOK
	}
	data.Selected = fileName
	_, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		data.DocumentError = message
		return data, status
	}
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

func (a *handler) fileNames() ([]string, error) {
	var names []string
	err := fs.WalkDir(a.source, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." || entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeType != 0 && !entry.Type().IsRegular() {
			return nil
		}
		if path.Ext(name) == ".cue" && validFileName(name) {
			names = append(names, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func (a *handler) readDocument(name string, knownFiles []string) ([]byte, cuebook.Document, int, string) {
	if !validFileName(name) || !containsFile(knownFiles, name) {
		return nil, cuebook.Document{}, http.StatusNotFound, "CUE file not found."
	}
	raw, err := fs.ReadFile(a.source, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, cuebook.Document{}, http.StatusNotFound, "CUE file not found."
		}
		return nil, cuebook.Document{}, http.StatusInternalServerError, "Unable to read this CUE file."
	}
	document, err := cuebook.New(raw)
	if err != nil {
		return raw, cuebook.Document{}, http.StatusUnprocessableEntity, "Unable to parse or validate this CUE document: " + err.Error()
	}
	return raw, document, http.StatusOK, ""
}

func validFileName(name string) bool {
	return name != "" && name != "." && fs.ValidPath(name) &&
		!strings.Contains(name, `\\`) && path.Ext(name) == ".cue"
}

func containsFile(names []string, name string) bool {
	index := sort.SearchStrings(names, name)
	return index < len(names) && names[index] == name
}

func fileURL(name string) string {
	query := url.Values{}
	query.Set("file", name)
	return "/?" + query.Encode()
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

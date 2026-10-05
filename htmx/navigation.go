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

type fileTreeNode struct {
	Name        string
	DisplayName string
	Path        string
	URL         string
	IsDir       bool
	Selected    bool
	Children    []fileTreeNode
}

type pageData struct {
	Files             []fileTreeNode
	Selected          string
	FileTitle         string
	FileDescription   string
	Entries           []entryView
	AddFields         []addFieldView
	RequiredAddFields []addFieldView
	OptionalAddFields []addFieldView
	AddError          string
	ReadOnly          bool
	Error             string
	DocumentError     string
}

type entryView struct {
	Index   int
	File    string
	Title   string
	CanMove bool
	Fields  []fieldView
	Details []fieldView
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
	data.Files = makeFileTree(fileNames, fileName)

	if fileName == "" {
		return data, http.StatusOK
	}
	data.Selected = fileName
	data.FileTitle = fileName
	raw, document, status, message := a.readDocument(fileName, fileNames)
	setFileFrontmatter(&data, fileName, raw)
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
			Index:   index,
			File:    fileName,
			Title:   entry.GetTitle(),
			CanMove: !readOnly,
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

func makeFileTree(fileNames []string, selected string) []fileTreeNode {
	var roots []fileTreeNode
	for _, fileName := range fileNames {
		parts := strings.Split(fileName, "/")
		children := &roots
		currentPath := ""
		for index, name := range parts {
			if currentPath == "" {
				currentPath = name
			} else {
				currentPath += "/" + name
			}
			isDir := index < len(parts)-1
			childIndex := -1
			for i := range *children {
				if (*children)[i].Name == name {
					childIndex = i
					break
				}
			}
			if childIndex == -1 {
				displayName := name
				if !isDir {
					displayName = strings.TrimSuffix(name, ".cue")
					if displayName == "" {
						displayName = name
					}
				}
				node := fileTreeNode{Name: name, DisplayName: displayName, Path: currentPath, IsDir: isDir}
				if !isDir {
					node.URL = fileURL(currentPath)
					node.Selected = currentPath == selected
				}
				*children = append(*children, node)
				childIndex = len(*children) - 1
			}
			children = &(*children)[childIndex].Children
		}
	}
	sortFileTree(roots)
	return roots
}

func sortFileTree(nodes []fileTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		left, right := strings.ToLower(nodes[i].Name), strings.ToLower(nodes[j].Name)
		if left == right {
			return nodes[i].Name < nodes[j].Name
		}
		return left < right
	})
	for i := range nodes {
		sortFileTree(nodes[i].Children)
	}
}

func (a *handler) pageForDocument(fileName string, fileNames []string, raw []byte, document cuebook.Document, notice string) (pageData, int) {
	data := a.basePage(fileNames, fileName, notice)
	data.Selected = fileName
	setFileFrontmatter(&data, fileName, raw)
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

func setFileFrontmatter(data *pageData, fileName string, source []byte) {
	data.FileTitle = fileName
	frontmatter := metadata.NewFrontmatter(source)
	if title := frontmatter.Title(); strings.TrimSpace(title) != "" {
		data.FileTitle = title
	}
	data.FileDescription = frontmatter.Description()
}

func (a *handler) basePage(fileNames []string, selected, notice string) pageData {
	return pageData{
		ReadOnly: a.committer == nil,
		Error:    notice,
		Files:    makeFileTree(fileNames, selected),
	}
}

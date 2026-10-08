package htmx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	markdownhtml "github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/util"
)

type pageData struct {
	Files             []fileTreeNode
	ArchiveFiles      []fileTreeNode
	ArchiveOpen       bool
	Selected          string
	FileTitle         string
	FileDescription   template.HTML
	Entries           []entryView
	SelectedEntry     *entryView
	AddFields         []addFieldView
	RequiredAddFields []addFieldView
	OptionalAddFields []addFieldView
	AddError          string
	ReadOnly          bool
	Error             string
	DocumentError     string
}

type entryView struct {
	Index     int
	File      string
	ItemURL   string
	Title     string
	CanMove   bool
	CanDelete bool
	Fields    []fieldView
	Details   []fieldView
}

type listRequest struct {
	File string `schema:"file"`
}

func (*listRequest) Validate(context.Context) error { return nil }

type listResponse struct {
	pageData
}

func (a *handler) list(_ context.Context, request *listRequest) (listResponse, error) {
	data, status := a.loadPage(request.File, "")
	response := listResponse{pageData: pageValues(data)}
	return response, responseErrorForStatus(status, pageResponseMessage(data))
}

func (a *handler) loadPage(fileName, notice string) (pageData, int) {
	data := pageData{ReadOnly: a.committer == nil, Error: notice}
	fileNames, err := a.fileNames()
	if err != nil {
		data.Error = "Unable to list CUE files."
		return data, http.StatusInternalServerError
	}
	data.Files, data.ArchiveFiles = makeSidebarTrees(fileNames, fileName)
	data.ArchiveOpen = strings.HasPrefix(fileName, archiveDirectory)

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

func makeEntryViews(document cuebook.Book, fileName string, readOnly bool) ([]entryView, error) {
	var result []entryView
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", index, err)
		}
		entryRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			return nil, fmt.Errorf("entry %d: unable to locate item in CUE file: %w", index, err)
		}
		view := makeEntryView(entry, fileName, index, entryRange, readOnly)
		for fieldIndex := range view.Fields {
			view.Fields[fieldIndex].ShowEditIcon = false
			view.Fields[fieldIndex].HideLabel = true
		}
		for fieldIndex := range view.Details {
			view.Details[fieldIndex].ShowEditIcon = false
			view.Details[fieldIndex].HideLabel = true
		}
		result = append(result, view)
		index++
	}
	return result, nil
}

func makeEntryView(entry cuebook.Entry, fileName string, index int, entryRange cuebook.ByteRange, readOnly bool) entryView {
	view := entryView{
		Index:     index,
		File:      fileName,
		ItemURL:   itemURL(fileName, entryRange),
		Title:     entry.GetTitle(),
		CanMove:   !readOnly,
		CanDelete: !readOnly && !strings.HasPrefix(fileName, archiveDirectory),
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
	return view
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

func (a *handler) readDocument(name string, knownFiles []string) ([]byte, cuebook.Book, int, string) {
	if !validFileName(name) || !containsFile(knownFiles, name) {
		return nil, cuebook.Book{}, http.StatusNotFound, "CUE file not found."
	}
	raw, err := fs.ReadFile(a.source, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, cuebook.Book{}, http.StatusNotFound, "CUE file not found."
		}
		return nil, cuebook.Book{}, http.StatusInternalServerError, "Unable to read this CUE file."
	}
	document, err := cuebook.New(raw)
	if err != nil {
		return raw, cuebook.Book{}, http.StatusUnprocessableEntity, "Unable to parse or validate this CUE document: " + err.Error()
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

func itemURL(fileName string, entryRange cuebook.ByteRange) string {
	query := url.Values{}
	query.Set("path", fileName)
	query.Set("head", strconv.Itoa(entryRange.Head))
	query.Set("tail", strconv.Itoa(entryRange.Tail))
	return "/item?" + query.Encode()
}

func (a *handler) pageForDocument(fileName string, fileNames []string, raw []byte, document cuebook.Book, notice string) (pageData, int) {
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

const (
	fileFrontmatterDetailsHead = `<remember-details data-storage-key="view-file-frontmatter"><summary>More...</summary><div id="file-frontmatter-description" class="file-description" data-details-content>`
	fileFrontmatterDetailsTail = `</div></remember-details>`
)

func setFileFrontmatter(data *pageData, fileName string, source []byte) {
	data.FileTitle = fileName
	descriptionParser := parser.New(parser.WithASTTransformers(util.Prioritized(
		metadata.NewDetailsTransformer(fileFrontmatterDetailsHead, fileFrontmatterDetailsTail), 0,
	)))
	frontmatter := metadata.NewFrontmatter(source, descriptionParser)
	if title := frontmatter.Title(); strings.TrimSpace(title) != "" {
		data.FileTitle = title
	}
	data.FileDescription = renderFrontmatterDescription(frontmatter)
}

func renderFrontmatterDescription(frontmatter metadata.Frontmatter) template.HTML {
	if frontmatter.Node == nil || frontmatter.Node.FirstChild() == nil {
		return ""
	}

	var rendered bytes.Buffer
	renderer := markdownhtml.New()
	for block := frontmatter.Node.FirstChild().NextSibling(); block != nil; block = block.NextSibling() {
		if rawHTML, ok := block.(*ast.HTMLBlock); ok && rawHTML.Value.IsOwned() {
			_, _ = rawHTML.Value.WriteTo(&rendered, frontmatter.Source)
			continue
		}
		if err := renderer.Render(&rendered, frontmatter.Source, block); err != nil {
			return template.HTML(template.HTMLEscapeString(frontmatter.Description()))
		}
	}
	return template.HTML(rendered.String())
}

func (a *handler) basePage(fileNames []string, selected, notice string) pageData {
	files, archiveFiles := makeSidebarTrees(fileNames, selected)
	return pageData{
		ReadOnly:     a.committer == nil,
		Error:        notice,
		Files:        files,
		ArchiveFiles: archiveFiles,
		ArchiveOpen:  strings.HasPrefix(selected, archiveDirectory),
	}
}

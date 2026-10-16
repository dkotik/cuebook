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
	gitview "github.com/dkotik/cuebook/htmx/git"
	"github.com/dkotik/cuebook/metadata"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	markdownhtml "github.com/yuin/goldmark/v2/renderer/html"
	"github.com/yuin/goldmark/v2/util"
)

func documentReadError(filePath string, status int, message string) error {
	if status == http.StatusNotFound {
		return &cuebook.FileNotFoundError{Path: filePath}
	}
	return errors.New(message)
}

type pageData struct {
	AgentEnabled      bool
	AgentBaseURL      string
	Files             []fileTreeNode
	ArchiveSelected   bool
	FolderView        bool
	FolderBreadcrumb  []folderBreadcrumbItem
	FolderItems       []fileTreeNode
	Selected          string
	FileTitle         string
	FileBreadcrumb    []fileBreadcrumbItem
	FileDescription   template.HTML
	Entries           []entryView
	SelectedEntry     *entryView
	AddFields         []addFieldView
	RequiredAddFields []addFieldView
	OptionalAddFields []addFieldView
	GitEnabled        bool
	GitSelected       bool
	GitLostLines      []string
	GitChanges        []gitview.FileChange

	ReadOnly      bool
	Error         string
	DocumentError string
}

type fileBreadcrumbItem struct {
	Label   string
	URL     string
	Current bool
}

type entryView struct {
	Index          int
	File           string
	EntryURL       string
	CUESource      string
	CutFingerprint string
	Title          string
	CanMove        bool
	CanArchive     bool
	MoveFilePicker filePickerView
	Fields         []fieldView
	Details        []fieldView
}

type listRequest struct {
	File   string `schema:"file"`
	Folder string `schema:"folder"`
}

func (*listRequest) Validate(context.Context) error { return nil }

type listResponse struct {
	pageData
}

func (a *handler) list(_ context.Context, request *listRequest) (listResponse, error) {
	var data pageData
	var status int
	fileOrFolder := request.File
	if request.File == "" {
		fileOrFolder = request.Folder
		data, status = a.loadFolderPage(request.Folder, "")
	} else {
		data, status = a.loadPage(request.File, "")
	}
	if status != http.StatusOK {
		message := data.Error
		if message == "" {
			message = data.DocumentError
		}
		return listResponse{}, documentReadError(fileOrFolder, status, message)
	}
	return listResponse{pageData: data}, nil
}

func (a *handler) loadFolderPage(folder, notice string) (pageData, int) {
	fileNames, err := a.fileNames()
	if err != nil {
		data := a.basePage(nil, folder, notice)
		data.Error = "Unable to list CUE files."
		return data, http.StatusInternalServerError
	}
	data := a.basePage(fileNames, folder, notice)
	items, ok := folderContents(makeFileTree(fileNames, ""), folder)
	if !ok {
		data.DocumentError = "Folder not found."
		return data, http.StatusNotFound
	}
	data.FolderView = true
	data.FolderBreadcrumb = makeFolderBreadcrumb(folder)
	data.FolderItems = items
	return data, http.StatusOK
}

func (a *handler) loadPage(fileName, notice string) (pageData, int) {
	data := pageData{
		ReadOnly:     a.committer == nil,
		Error:        notice,
		AgentEnabled: a.agentEnabled,
		AgentBaseURL: a.agentBaseURL,
		GitEnabled:   a.gitFileLinker != nil,
	}
	fileNames, err := a.fileNames()
	if err != nil {
		data.Error = "Unable to list CUE files."
		return data, http.StatusInternalServerError
	}
	data.Files = makeSidebarTree(fileNames, fileName)
	data.ArchiveSelected = fileName == archiveFolder || strings.HasPrefix(fileName, archiveDirectory)

	if fileName == "" {
		return data, http.StatusOK
	}
	data.Selected = fileName
	raw, document, status, message := a.readDocument(fileName, fileNames)
	setFileFrontmatter(&data, raw)
	data.FileBreadcrumb = makeFileBreadcrumb(fileName)
	if status != http.StatusOK {
		data.DocumentError = message
		return data, status
	}
	if !data.ReadOnly {
		data.AddFields = makeAddFieldViews(document)
		data.RequiredAddFields, data.OptionalAddFields = splitAddFieldViews(data.AddFields)

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
			return nil, fmt.Errorf("entry %d: unable to locate entry in CUE file: %w", index, err)
		}
		view := makeEntryView(entry, fileName, index, entryRange, readOnly, false)
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

func makeEntryView(entry cuebook.Entry, fileName string, index int, entryRange cuebook.ByteRange, readOnly, includeTitleField bool) entryView {
	view := entryView{
		Index:      index,
		File:       fileName,
		EntryURL:   entryURL(fileName, entryRange),
		Title:      entry.GetTitle(),
		CanMove:    !readOnly,
		CanArchive: !readOnly && !strings.HasPrefix(fileName, archiveDirectory),
	}
	if view.Title == "" {
		view.Title = fmt.Sprintf("Entry %d", index+1)
	}
	hasTitleField := false
	if !includeTitleField {
		for _, field := range entry.Fields {
			hasTitleField = hasTitleField || metadata.IsTitleField(field.Value)
		}
		for _, field := range entry.Details {
			hasTitleField = hasTitleField || metadata.IsTitleField(field.Value)
		}
	}
	for fieldIndex, field := range entry.Fields {
		if !includeTitleField && (metadata.IsTitleField(field.Value) || (!hasTitleField && fieldIndex == 0 && entry.GetTitle() != "")) {
			continue
		}
		fieldView := makeFieldView(field, fileName, index, readOnly, includeTitleField)
		fieldView.Optional = entryFieldOptional(entry.Value, field.Name)
		view.Fields = append(view.Fields, fieldView)
	}
	for _, field := range entry.Details {
		if !includeTitleField && metadata.IsTitleField(field.Value) {
			continue
		}
		fieldView := makeFieldView(field, fileName, index, readOnly, includeTitleField)
		fieldView.Optional = entryFieldOptional(entry.Value, field.Name)
		view.Details = append(view.Details, fieldView)
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

func entryURL(fileName string, entryRange cuebook.ByteRange) string {
	query := url.Values{}
	query.Set("path", fileName)
	query.Set("head", strconv.Itoa(entryRange.Head))
	query.Set("tail", strconv.Itoa(entryRange.Tail))
	return "/entry?" + query.Encode()
}

const (
	fileFrontmatterDetailsHead = `<remember-details data-storage-key="view-file-frontmatter"><summary>More...</summary><div id="file-frontmatter-description" class="file-description" data-details-content>`
	fileFrontmatterDetailsTail = `</div></remember-details>`
)

func makeFileBreadcrumb(fileName string) []fileBreadcrumbItem {
	parts := strings.Split(fileName, "/")
	breadcrumb := make([]fileBreadcrumbItem, 0, len(parts))
	for index, part := range parts {
		item := fileBreadcrumbItem{Label: part, Current: index == len(parts)-1}
		folder := strings.Join(parts[:index+1], "/")
		if item.Current {
			item.Label = strings.TrimSuffix(part, path.Ext(part))
			if item.Label == "" {
				item.Label = part
			}
			folder = path.Dir(fileName)
			if folder == "." {
				folder = ""
			}
		}
		item.URL = folderURL(folder)
		breadcrumb = append(breadcrumb, item)
	}
	return breadcrumb
}

func setFileFrontmatter(data *pageData, source []byte) {
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
	return pageData{
		AgentEnabled:    a.agentEnabled,
		AgentBaseURL:    a.agentBaseURL,
		GitEnabled:      a.gitFileLinker != nil,
		ReadOnly:        a.committer == nil,
		Error:           notice,
		Files:           makeSidebarTree(fileNames, selected),
		ArchiveSelected: selected == archiveFolder || strings.HasPrefix(selected, archiveDirectory),
	}
}

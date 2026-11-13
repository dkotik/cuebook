package htmx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/dkotik/cuebook"
	gitview "github.com/dkotik/cuebook/htmx/git"
	"github.com/dkotik/cuebook/internal/markdown"
	"github.com/dkotik/cuebook/metadata"
	"github.com/yuin/goldmark/v2/ast"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/util"
)

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
	UUID           string
	File           string
	FileURL        string
	EntryURL       string
	CUESource      string
	CutFingerprint string
	Title          string
	CanMove        bool
	CanArchive     bool
	CanAddField    bool
	NewFieldKinds  []newFieldKindView
	MoveFilePicker filePickerView
	Fields         []fieldView
	Details        []fieldView
}

type listRequest struct {
	File     string `schema:"file"`
	Filepath string `schema:"filepath"`
	Folder   string `schema:"folder"`
}

func (*listRequest) Validate(context.Context) error { return nil }

type listResponse struct {
	pageData
}

func (a *handler) list(_ context.Context, request *listRequest) (listResponse, error) {
	var data pageData
	var err error
	fileName := request.Filepath
	if fileName == "" {
		fileName = request.File
	}
	if fileName == "" {
		data, err = a.loadFolderPage(request.Folder, "")
	} else {
		data, err = a.loadPage(fileName, "")
	}
	if err != nil {
		return listResponse{}, err
	}
	return listResponse{pageData: data}, nil
}

func (a *handler) loadFolderPage(folder, notice string) (pageData, error) {
	fileNames, err := a.fileNames()
	if err != nil {
		data := a.basePage(nil, folder, notice)
		data.Error = ErrUnableToListCueFiles.Error()
		return data, ErrUnableToListCueFiles
	}
	data := a.basePage(fileNames, folder, notice)
	items, ok := folderContents(makeFileTree(fileNames, ""), folder)
	if !ok {
		data.DocumentError = "Folder not found."
		return data, &cuebook.FileSystemPathError{PathError: &fs.PathError{
			Path: folder, Err: cuebook.ErrFileSystemFileNotFound,
		}}
	}
	data.FolderView = true
	data.FolderBreadcrumb = makeFolderBreadcrumb(folder)
	data.FolderItems = items
	return data, nil
}

func (a *handler) loadPage(fileName, notice string) (pageData, error) {
	data := pageData{
		ReadOnly:     a.files == nil,
		Error:        notice,
		AgentEnabled: a.agentEnabled,
		AgentBaseURL: a.agentBaseURL,
		GitEnabled:   a.gitFileLinker != nil,
	}
	fileNames, err := a.fileNames()
	if err != nil {
		data.Error = ErrUnableToListCueFiles.Error()
		return data, ErrUnableToListCueFiles
	}
	data.Files = makeSidebarTree(fileNames, fileName)
	data.ArchiveSelected = fileName == archiveFolder || strings.HasPrefix(fileName, archiveDirectory)

	if fileName == "" {
		return data, nil
	}
	data.Selected = fileName
	raw, document, err := a.readDocument(fileName, fileNames)
	setFileFrontmatter(&data, raw)
	data.FileBreadcrumb = makeFileBreadcrumb(fileName)
	if err != nil {
		data.DocumentError = err.Error()
		return data, err
	}
	if !data.ReadOnly {
		data.AddFields = makeAddFieldViews(document)
		data.RequiredAddFields, data.OptionalAddFields = splitAddFieldViews(data.AddFields)
	}
	entries, err := makeEntryViews(document, fileName, data.ReadOnly)
	if err != nil {
		data.DocumentError = "Unable to display this CUE document: " + err.Error()
		return data, fmt.Errorf("Unable to display this CUE document: %w", err)
	}
	data.Entries = entries
	return data, nil
}

func makeEntryViews(document cuebook.Book, fileName string, readOnly bool) ([]entryView, error) {
	var result []entryView
	index := 0
	for entry, err := range document.EachEntry() {
		if err != nil {
			return nil, &cuebook.FileSystemPathError{PathError: &fs.PathError{
				Op: "read", Path: fileName, Err: err,
			}}
		}
		entryRange, err := cuebook.NewByteRange(entry.Value)
		if err != nil {
			return nil, &cuebook.FileSystemPathError{PathError: &fs.PathError{
				Op: "read", Path: fileName, Err: err,
			}}
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
		Index:       index,
		UUID:        entry.UUID,
		File:        fileName,
		FileURL:     fileURL(fileName),
		EntryURL:    entryURLForEntry(fileName, entryRange, entry.UUID),
		Title:       entry.Title,
		CanMove:     !readOnly,
		CanArchive:  !readOnly && !strings.HasPrefix(fileName, archiveDirectory),
		CanAddField: !readOnly && metadata.IsOpen(entry.Value),
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
		if !includeTitleField && (metadata.IsTitleField(field.Value) || (!hasTitleField && fieldIndex == 0 && entry.Title != "")) {
			continue
		}
		fieldView := makeFieldView(field, fileName, index, readOnly, includeTitleField, entry.UUID)
		fieldView.Optional = entryFieldOptional(entry.Value, field.Name)
		view.Fields = append(view.Fields, fieldView)
	}
	for _, field := range entry.Details {
		if !includeTitleField && metadata.IsTitleField(field.Value) {
			continue
		}
		fieldView := makeFieldView(field, fileName, index, readOnly, includeTitleField, entry.UUID)
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

func (a *handler) readDocument(name string, knownFiles []string) ([]byte, cuebook.Book, error) {
	fileNotFound := &cuebook.FileSystemPathError{PathError: &fs.PathError{
		Path: name, Err: cuebook.ErrFileSystemFileNotFound,
	}}
	if !validFileName(name) || !containsFile(knownFiles, name) {
		return nil, cuebook.Book{}, fileNotFound
	}
	raw, err := fs.ReadFile(a.source, name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, cuebook.Book{}, fileNotFound
		}
		var pathError *fs.PathError
		if errors.As(err, &pathError) {
			return nil, cuebook.Book{}, &cuebook.FileSystemPathError{PathError: pathError}
		}
		return nil, cuebook.Book{}, &cuebook.FileSystemPathError{PathError: &fs.PathError{
			Op: "read", Path: name, Err: err,
		}}
	}
	document, err := cuebook.New(raw)
	if err != nil {
		return raw, cuebook.Book{}, fmt.Errorf("Unable to parse or validate this CUE document: %w", err)
	}
	return raw, document, nil
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
	return entryURLForEntry(fileName, entryRange, "")
}

func entryURLForEntry(fileName string, entryRange cuebook.ByteRange, uuid string) string {
	query := url.Values{}
	filePath, hasFileRoute := cueFileRoutePath(fileName)
	entryPath := "/entry"
	if uuid != "" {
		query.Set("uuid", uuid)
		entryPath += "/uuid"
	} else {
		query.Set("head", strconv.Itoa(entryRange.Head))
		query.Set("tail", strconv.Itoa(entryRange.Tail))
	}
	if hasFileRoute {
		entryPath += filePath
	} else {
		query.Set("path", fileName)
	}
	return entryPath + "?" + query.Encode()
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
	descriptionParser := markdown.NewParser(parser.WithASTTransformers(util.Prioritized(
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
	renderer := markdown.NewRenderer()
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
		ReadOnly:        a.files == nil,
		Error:           notice,
		Files:           makeSidebarTree(fileNames, selected),
		ArchiveSelected: selected == archiveFolder || strings.HasPrefix(selected, archiveDirectory),
	}
}

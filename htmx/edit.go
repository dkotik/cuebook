package htmx

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"cuelang.org/go/cue"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/patch"
)

type editFormRequest struct {
	File  string `schema:"file"`
	Entry string `schema:"entry"`
	Field string `schema:"field"`
	Mode  string `schema:"mode"`
}

func (*editFormRequest) Validate(context.Context) error {
	return nil
}

type editFormResponse struct {
	fieldView
	templateName string
}

func (a *handler) editForm(_ context.Context, request *editFormRequest) (editFormResponse, error) {
	if a.committer == nil {
		return editFormResponse{}, errors.New("This source is read-only.")
	}

	fileNames, err := a.fileNames()
	if err != nil {
		return editFormResponse{}, errors.New("Unable to list CUE files.")
	}
	_, document, status, message := a.readDocument(request.File, fileNames)
	if status != http.StatusOK {
		return editFormResponse{}, documentReadError(request.File, status, message)
	}

	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil {
		return editFormResponse{}, errors.New("The entry index is invalid.")
	}
	if entryIndex < 0 {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	entryValue, err := document.GetValue(entryIndex)
	if err != nil {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		return editFormResponse{}, errors.New("Unable to read this entry.")
	}
	field, ok := entry.GetFieldByName(request.Field)
	if !ok || request.Field == "" {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}

	view := makeFieldView(field, request.File, entryIndex, false)
	templateName := "field-form"
	if request.Mode == "view" {
		templateName = "field"
	}
	response := editFormResponse{fieldView: view, templateName: templateName}
	return response, nil
}

type editRequest struct {
	File  string  `schema:"file"`
	Entry string  `schema:"entry"`
	Field string  `schema:"field"`
	Value *string `schema:"value"`
}

func (*editRequest) Validate(context.Context) error { return nil }

type editResponse struct {
	pageData
	redirect string
}

func (response editResponse) GetRedirect() string {
	return response.redirect
}

type editErrorResponse struct {
	pageData
}

func (response editErrorResponse) GetFlashMessage() string {
	return response.Error
}

func (a *handler) edit(_ context.Context, request *editRequest) (editResponse, error) {
	if a.committer == nil {
		return editResponse{}, errors.New("This source is read-only.")
	}
	if request.Value == nil {
		return editResponse{}, errors.New("The edit request is invalid.")
	}
	value := *request.Value

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return editResponse{}, errors.New("Unable to list CUE files.")
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return editResponse{}, documentReadError(fileName, status, message)
	}

	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil {
		return editResponse{}, errors.New("The entry index is invalid.")
	}
	if entryIndex < 0 {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entryValue, err := document.GetValue(entryIndex)
	if err != nil {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		return editResponse{}, errors.New("Unable to read this entry.")
	}
	fieldName := request.Field
	field, ok := entry.GetFieldByName(fieldName)
	if !ok || fieldName == "" {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	if isSecretField(field.Value) && value == "" && field.String() != "" {
		data, err := a.renderEditedFile(fileName)
		return editResponse{pageData: data, redirect: a.fileListURL(fileName)}, err
	}

	change, err := patch.UpdateFieldValue(raw, entryValue, field.Value, value)
	if err != nil {
		return editResponse{}, errors.New("The field value could not be formatted.")
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return editResponse{}, errors.New("The entry changed before the edit could be applied. Reload and try again.")
	}
	if _, err = cuebook.New(candidate); err != nil {
		return editResponse{}, errors.New("The submitted value does not satisfy the CUE constraints: " + err.Error())
	}
	if err = a.commitFile(fileName, change); err != nil {
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			return editResponse{}, errors.New("The entry changed before the edit could be applied. Reload and try again.")
		}
		return editResponse{}, errors.New("The edit could not be saved.")
	}
	data, err := a.renderEditedFile(fileName)
	return editResponse{pageData: data, redirect: a.fileListURL(fileName)}, err
}

func (a *handler) fileListURL(fileName string) string {
	path := routeWithPrefix(a.routePrefix, "")
	if path == "" {
		path = "/"
	} else if path != "/" {
		path += "/"
	}
	query := url.Values{"file": {fileName}}
	return path + "?" + query.Encode()
}

func (a *handler) renderEditedFile(fileName string) (pageData, error) {
	data, status := a.loadPage(fileName, "")
	if status == http.StatusNotFound {
		return pageData{}, &cuebook.FileNotFoundError{Path: fileName}
	}
	if status != http.StatusOK {
		return pageData{}, errors.New("The edit was saved, but the updated document could not be displayed.")
	}
	return data, nil
}

func isSecretField(field cue.Value) bool {
	_, ok := metadata.GetFieldAttributes(field, "cuebook").GetFirstOf("argon2id")
	return ok
}

type fieldView struct {
	File         string
	Index        int
	Name         string
	Description  string
	Value        string
	EditURL      string
	ViewURL      string
	MultiLine    bool
	Secret       bool
	ReadOnly     bool
	Editing      bool
	ShowEditIcon bool
	HideLabel    bool
}

func fieldEditURL(fileName string, entryIndex int, fieldName string, view bool) string {
	query := url.Values{}
	query.Set("entry", strconv.Itoa(entryIndex))
	query.Set("field", fieldName)
	query.Set("file", fileName)
	if view {
		query.Set("mode", "view")
	}
	return "/edit?" + query.Encode()
}

func makeFieldView(field cuebook.Field, fileName string, index int, readOnly bool) fieldView {
	_, secret := metadata.GetFieldAttributes(field.Value, "cuebook").GetFirstOf("argon2id")
	return fieldView{
		File:         fileName,
		Index:        index,
		Name:         field.Name,
		Description:  field.Description,
		Value:        field.String(),
		EditURL:      fieldEditURL(fileName, index, field.Name, false),
		ViewURL:      fieldEditURL(fileName, index, field.Name, true),
		MultiLine:    metadata.IsMultiLine(field.Value),
		Secret:       secret,
		ReadOnly:     readOnly,
		ShowEditIcon: true,
	}
}

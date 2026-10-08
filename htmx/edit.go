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

func editFormFailure(status int, message string) (editFormResponse, error) {
	return editFormResponse{}, responseErrorForStatus(status, message)
}

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
		return editFormFailure(http.StatusForbidden, "This source is read-only.")
	}

	fileNames, err := a.fileNames()
	if err != nil {
		return editFormFailure(http.StatusInternalServerError, "Unable to list CUE files.")
	}
	_, document, status, message := a.readDocument(request.File, fileNames)
	if status != http.StatusOK {
		return editFormFailure(status, message)
	}

	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil || entryIndex < 0 {
		return editFormFailure(http.StatusBadRequest, "The entry index is invalid.")
	}
	entryValue, err := document.GetValue(entryIndex)
	if err != nil {
		return editFormFailure(http.StatusNotFound, "Entry not found.")
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		return editFormFailure(http.StatusUnprocessableEntity, "Unable to read this entry.")
	}
	field, ok := entry.GetFieldByName(request.Field)
	if !ok || request.Field == "" {
		return editFormFailure(http.StatusNotFound, "Field not found.")
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
}

func (a *handler) edit(ctx context.Context, request *editRequest) (editResponse, error) {
	if a.committer == nil {
		return editResponseFrom(a.editFailure(ctx, "", "This source is read-only.", http.StatusForbidden))
	}
	if request.Value == nil {
		return editResponseFrom(a.editFailure(ctx, request.File, "The edit request is invalid.", http.StatusBadRequest))
	}
	value := *request.Value

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return editResponseFrom(a.editFailure(ctx, "", "Unable to list CUE files.", http.StatusInternalServerError))
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return editResponseFrom(a.editFailure(ctx, fileName, message, status))
	}
	page, _ := a.pageForDocument(fileName, fileNames, raw, document, "")

	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil || entryIndex < 0 {
		return editResponseFrom(withNotice(page, "The entry index is invalid."), http.StatusBadRequest)
	}
	entryValue, err := document.GetValue(entryIndex)
	if err != nil {
		return editResponseFrom(withNotice(page, "Entry not found."), http.StatusNotFound)
	}
	entry, err := cuebook.NewEntry(entryValue)
	if err != nil {
		return editResponseFrom(withNotice(page, "Unable to read this entry."), http.StatusUnprocessableEntity)
	}
	fieldName := request.Field
	field, ok := entry.GetFieldByName(fieldName)
	if !ok || fieldName == "" {
		return editResponseFrom(withNotice(page, "Field not found."), http.StatusNotFound)
	}
	if isSecretField(field.Value) && value == "" && field.String() != "" {
		return editResponseFrom(a.finishEdit(fileName))
	}

	change, err := patch.UpdateFieldValue(raw, entryValue, field.Value, value)
	if err != nil {
		return editResponseFrom(a.renderEditInputFailure(ctx, page, entryIndex, fieldName, value, "The field value could not be formatted.", http.StatusUnprocessableEntity))
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return editResponseFrom(a.renderEditInputFailure(ctx, page, entryIndex, fieldName, value, "The entry changed before the edit could be applied. Reload and try again.", http.StatusConflict))
	}
	if _, err = cuebook.New(candidate); err != nil {
		return editResponseFrom(a.renderEditInputFailure(ctx, page, entryIndex, fieldName, value, "The submitted value does not satisfy the CUE constraints: "+err.Error(), http.StatusUnprocessableEntity))
	}
	if err = a.commitFile(fileName, change); err != nil {
		status = http.StatusInternalServerError
		notice := "The edit could not be saved."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The entry changed before the edit could be applied. Reload and try again."
		}
		return editResponseFrom(a.editFailure(ctx, fileName, notice, status))
	}
	return editResponseFrom(a.finishEdit(fileName))
}

func (a *handler) renderEditInputFailure(_ context.Context, page pageData, entryIndex int, fieldName, value, notice string, status int) (pageData, int) {
	for i := range page.Entries {
		entry := &page.Entries[i]
		if entry.Index != entryIndex {
			continue
		}
		if markEditingField(entry.Fields, fieldName, value) || markEditingField(entry.Details, fieldName, value) {
			break
		}
	}
	return withNotice(page, notice), status
}

func markEditingField(fields []fieldView, fieldName, value string) bool {
	for i := range fields {
		if fields[i].Name != fieldName {
			continue
		}
		fields[i].Editing = true
		fields[i].Value = value
		return true
	}
	return false
}

func (a *handler) finishEdit(fileName string) (pageData, int) {
	return a.renderEditedFile(fileName)
}

func (a *handler) renderEditedFile(fileName string) (pageData, int) {
	data, status := a.loadPage(fileName, "")
	if status != http.StatusOK {
		data.Error = "The edit was saved, but the updated document could not be displayed."
		status = http.StatusInternalServerError
	}
	return data, status
}

func (a *handler) editFailure(_ context.Context, fileName, notice string, status int) (pageData, int) {
	data, _ := a.loadPage(fileName, notice)
	return data, status
}

func isSecretField(field cue.Value) bool {
	_, ok := metadata.GetFieldAttributes(field, "cuebook").GetFirstOf("argon2id")
	return ok
}

func withNotice(data pageData, notice string) pageData {
	data.Error = notice
	return data
}

type fieldView struct {
	File         string
	Index        int
	Name         string
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
		Value:        field.String(),
		EditURL:      fieldEditURL(fileName, index, field.Name, false),
		ViewURL:      fieldEditURL(fileName, index, field.Name, true),
		MultiLine:    metadata.IsMultiLine(field.Value),
		Secret:       secret,
		ReadOnly:     readOnly,
		ShowEditIcon: true,
	}
}

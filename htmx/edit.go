package htmx

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/token"
	"github.com/dkotik/cuebook"
	fieldmodel "github.com/dkotik/cuebook/htmx/field"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/patch"
)

type editFormRequest struct {
	File  string `schema:"file"`
	Entry string `schema:"entry"`
	Field string `schema:"field"`
	Mode  string `schema:"mode"`
	View  string `schema:"view"`
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

	view := makeFieldView(field, request.File, entryIndex, false, request.View == "entry")
	view.Optional = entryFieldOptional(entryValue, field.Name)
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
	View  string  `schema:"view"`
}

func (*editRequest) Validate(context.Context) error { return nil }

type editResponse struct {
	pageData
	destination string
}

func (response editResponse) GetURLReplacement() string {
	return response.destination
}

type editErrorResponse struct {
	pageData
}

func (response editErrorResponse) GetFlashMessage() (string, bool) {
	return response.Error, response.Error == ""
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
		return a.renderEditResponse(request, entryIndex)
	}

	unset := value == "not_set" && field.Value.IncompleteKind() == cue.BoolKind && !isSecretField(field.Value)
	var change patch.Patch
	if unset {
		change, err = patch.RemoveFieldValue(raw, entryValue, fieldName)
	} else {
		change, err = patch.UpdateFieldValue(raw, entryValue, field.Value, value)
	}
	if err != nil {
		return editResponse{}, errors.New("The field value could not be formatted.")
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return editResponse{}, errors.New("The entry changed before the edit could be applied. Reload and try again.")
	}
	if unset {
		// Inspect the field after omission: a populated optional field has a regular
		// selector, and a required field with a default may still validate.
		omitted := cuecontext.New().CompileBytes(candidate).LookupPath(cue.MakePath(cue.Index(entryIndex)))
		if !optionalField(omitted, fieldName) {
			return editResponse{}, errors.New("The required boolean field " + fieldName + " cannot be unset.")
		}
		if !entryValue.LookupPath(cue.MakePath(cue.Str(fieldName))).Exists() {
			return a.renderEditResponse(request, entryIndex)
		}
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
	return a.renderEditResponse(request, entryIndex)
}

func (a *handler) renderEditResponse(request *editRequest, entryIndex int) (editResponse, error) {
	if request.View == "entry" {
		entry, err := a.loadEntryPage(request.File, cuebook.ByteRange{}, entryIndex)
		if err != nil {
			return editResponse{}, err
		}
		return editResponse{
			pageData:    entry.pageData,
			destination: routeWithPrefix(a.routePrefix, entry.EntryURL),
		}, nil
	}
	data, err := a.renderEditedFile(request.File)
	return editResponse{pageData: data}, err
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

// entryFieldOptional also checks inherited constraints: populated optional
// fields have regular selectors on the unified entry, but not on its schema.
func entryFieldOptional(entry cue.Value, name string) bool {
	if optionalField(entry, name) {
		return true
	}
	if op, operands := entry.Expr(); op == cue.AndOp {
		// A regular data field does not make an optional schema field required,
		// but a regular field inherited from another list constraint does.
		for _, operand := range operands {
			if _, inherited := operand.Source().(*ast.Ellipsis); inherited &&
				operand.LookupPath(cue.MakePath(cue.Str(name))).Exists() && !optionalField(operand, name) {
				return false
			}
		}
		for _, operand := range operands {
			if entryFieldOptional(operand, name) {
				return true
			}
		}
	}
	return false
}

func optionalField(entry cue.Value, name string) bool {
	fields, err := entry.Fields(cue.Optional(true))
	if err != nil {
		return false
	}
	for fields.Next() {
		if fields.Selector().Unquoted() == name {
			return fields.Selector().ConstraintType() == cue.OptionalConstraint
		}
	}
	return false
}

type fieldView struct {
	File         string
	Index        int
	Name         string
	Description  string
	Value        string
	Empty        bool
	EditURL      string
	ViewURL      string
	MultiLine    bool
	Boolean      bool
	Optional     bool
	Secret       bool
	ReadOnly     bool
	Editing      bool
	EntryView    bool
	ShowEditIcon bool
	HideLabel    bool
}

func fieldEditURL(fileName string, entryIndex int, fieldName string, view, entryView bool) string {
	query := url.Values{}
	query.Set("entry", strconv.Itoa(entryIndex))
	query.Set("field", fieldName)
	query.Set("file", fileName)
	if view {
		query.Set("mode", "view")
	}
	if entryView {
		query.Set("view", "entry")
	}
	return "/edit?" + query.Encode()
}

// optionalFieldValue inspects declarations because CUE value paths do not retain
// optional selectors. Every conjunct must be optional for the field to be absent.
func optionalFieldValue(value cue.Value) bool {
	if source, ok := value.Source().(*ast.Field); ok {
		return source.Constraint == token.OPTION
	}
	if op, operands := value.Expr(); op == cue.AndOp && len(operands) > 0 {
		for _, operand := range operands {
			if !optionalFieldValue(operand) {
				return false
			}
		}
		return true
	}
	return false
}

func makeFieldView(field fieldmodel.Field, fileName string, index int, readOnly, entryView bool) fieldView {
	_, secret := metadata.GetFieldAttributes(field.Value, "cuebook").GetFirstOf("argon2id")
	booleanOnly := field.Value.IncompleteKind() == cue.BoolKind
	value := field.String()
	concrete := field.Value.IsConcrete()
	if boolean, err := field.Value.Bool(); err == nil {
		value = strconv.FormatBool(boolean)
		// Bool resolves defaults even when IsConcrete is false. Honor that value
		// for regular fields, but never materialize an absent optional default.
		if booleanOnly && !secret && !optionalFieldValue(field.Value) {
			concrete = true
		}
	}
	empty := !concrete || strings.TrimSpace(value) == ""
	if !concrete {
		value = ""
		if booleanOnly && !secret {
			value = "not_set"
		}
	}
	return fieldView{
		File:         fileName,
		Index:        index,
		Name:         field.Name,
		Description:  field.Description,
		Value:        value,
		Empty:        empty,
		EditURL:      fieldEditURL(fileName, index, field.Name, false, entryView),
		ViewURL:      fieldEditURL(fileName, index, field.Name, true, entryView),
		EntryView:    entryView,
		MultiLine:    metadata.IsMultiLine(field.Value),
		Boolean:      booleanOnly,
		Secret:       secret,
		ReadOnly:     readOnly,
		ShowEditIcon: true,
	}
}

package htmx

import (
	"context"
	"errors"
	"html/template"
	"io/fs"
	"net/url"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/token"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
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
	if a.files == nil {
		return editFormResponse{}, ErrSourceReadOnly
	}

	fileNames, err := a.fileNames()
	if err != nil {
		return editFormResponse{}, ErrUnableToListCueFiles
	}
	_, document, err := a.readDocument(request.File, fileNames)
	if err != nil {
		return editFormResponse{}, err
	}

	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil {
		return editFormResponse{}, ErrInvalidEntryIndex
	}
	if entryIndex < 0 {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	entry, err := document.GetEntry(entryIndex)
	if err != nil {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	entryValue := entry.Value
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

func (a *handler) edit(ctx context.Context, request *editRequest) (editResponse, error) {
	if a.files == nil {
		return editResponse{}, ErrSourceReadOnly
	}
	if request.Value == nil {
		return editResponse{}, ErrInvalidEditRequest
	}
	value := *request.Value

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return editResponse{}, ErrUnableToListCueFiles
	}
	raw, document, err := a.readDocument(fileName, fileNames)
	if err != nil {
		return editResponse{}, err
	}

	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil {
		return editResponse{}, ErrInvalidEntryIndex
	}
	if entryIndex < 0 {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entry, err := document.GetEntry(entryIndex)
	if err != nil {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entryValue := entry.Value
	fieldName := request.Field
	field, ok := entry.GetFieldByName(fieldName)
	if !ok || fieldName == "" {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	if metadata.IsLockedField(field.Value) {
		return editResponse{}, FieldError{Code: ErrFieldLocked, Field: fieldName}
	}
	if isSecretField(field.Value) && value == "" && field.String() != "" {
		return a.renderEditResponse(request, entryIndex)
	}

	unset := value == "not_set" && field.Value.IncompleteKind() == cue.BoolKind && !isSecretField(field.Value)
	var change cuebook.Patch
	if unset {
		change, err = entry.RemoveFieldValue(fieldName)
	} else {
		change, err = entry.UpdateFieldValue(fieldName, value)
	}
	if err != nil {
		return editResponse{}, CauseError{Code: ErrFieldValueFormatFailed, Cause: err}
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return editResponse{}, CauseError{Code: ErrDocumentChangedBeforeEdit, Cause: err}
	}
	if unset {
		// Inspect the field after omission: a populated optional field has a regular
		// selector, and a required field with a default may still validate.
		omitted := cuecontext.New().CompileBytes(candidate).LookupPath(cue.MakePath(cue.Index(entryIndex)))
		if !optionalField(omitted, fieldName) {
			return editResponse{}, FieldError{Code: ErrRequiredBooleanFieldUnset, Field: fieldName}
		}
		if !entryValue.LookupPath(cue.MakePath(cue.Str(fieldName))).Exists() {
			return a.renderEditResponse(request, entryIndex)
		}
	}
	if _, err = cuebook.New(candidate); err != nil {
		return editResponse{}, CauseError{Code: ErrSubmittedValueConstraintViolation, Cause: err}
	}
	if err = a.commitFile(ctx, fileName, change); err != nil {
		return editResponse{}, err
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
	data, err := a.loadPage(fileName, "")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return pageData{}, err
		}
		return pageData{}, ErrEditSavedButDisplayFailed
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
	Locked       bool
	ReadOnly     bool
	Editing      bool
	EntryView    bool
	ShowEditIcon bool
	HideLabel    bool
	Markdown     template.HTML
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

func makeFieldView(field cuebook.Field, fileName string, index int, readOnly, entryView bool) fieldView {
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
	var markdown template.HTML
	if metadata.IsMultiLine(field.Value) && !secret && !empty {
		markdown = renderFieldMarkdown(value)
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
		Locked:       metadata.IsLockedField(field.Value),
		ReadOnly:     readOnly,
		ShowEditIcon: true,
		Markdown:     markdown,
	}
}

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

func (*editFormRequest) Validate(context.Context) error { return nil }

type editUUIDFormRequest struct {
	File  string `schema:"file"`
	UUID  string `schema:"uuid"`
	Field string `schema:"field"`
	Mode  string `schema:"mode"`
	View  string `schema:"view"`
}

func (*editUUIDFormRequest) Validate(context.Context) error { return nil }

type editFormResponse struct {
	fieldView
	templateName string
}

func (a *handler) editForm(_ context.Context, request *editFormRequest) (editFormResponse, error) {
	if a.files == nil {
		return editFormResponse{}, ErrSourceReadOnly
	}
	_, _, document, err := a.loadEntryDocument(request.File)
	if err != nil {
		return editFormResponse{}, err
	}
	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil {
		return editFormResponse{}, ErrInvalidEntryIndex
	}
	return a.editFormAtIndex(request.File, document, entryIndex, request.Field, request.Mode, request.View)
}

func (a *handler) editFormUUID(_ context.Context, request *editUUIDFormRequest) (editFormResponse, error) {
	if a.files == nil {
		return editFormResponse{}, ErrSourceReadOnly
	}
	_, _, document, err := a.loadEntryDocument(request.File)
	if err != nil {
		return editFormResponse{}, err
	}
	entryIndex, err := entryIndexByUUID(document, request.UUID)
	if errors.Is(err, cuebook.ErrBookEntryNotFound) {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	if err != nil {
		return editFormResponse{}, err
	}
	return a.editFormAtIndex(request.File, document, entryIndex, request.Field, request.Mode, request.View)
}

func (a *handler) editFormAtIndex(fileName string, document cuebook.Book, entryIndex int, fieldName, mode, view string) (editFormResponse, error) {
	entry, err := document.GetEntryByIndex(entryIndex)
	if err != nil {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	field, ok := entry.GetFieldByName(fieldName)
	if !ok || fieldName == "" {
		return editFormResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}

	fieldData := makeFieldView(field, fileName, entryIndex, false, view == "entry", entry.UUID)
	fieldData.Optional = entryFieldOptional(entry.Value, field.Name)
	templateName := "field-form"
	if mode == "view" {
		templateName = "field"
	}
	return editFormResponse{fieldView: fieldData, templateName: templateName}, nil
}

type editRequest struct {
	File  string  `schema:"file"`
	Entry string  `schema:"entry"`
	Field string  `schema:"field"`
	Value *string `schema:"value"`
	View  string  `schema:"view"`
}

func (*editRequest) Validate(context.Context) error { return nil }

type editUUIDRequest struct {
	File  string  `schema:"file"`
	UUID  string  `schema:"uuid"`
	Field string  `schema:"field"`
	Value *string `schema:"value"`
	View  string  `schema:"view"`
}

func (*editUUIDRequest) Validate(context.Context) error { return nil }

type editInput struct {
	Field string
	Value *string
	View  string
}

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

type entryPageRenderer func() (entryResponse, error)

func (a *handler) edit(ctx context.Context, request *editRequest) (editResponse, error) {
	raw, document, err := a.loadEditableDocument(request.File, request.Value)
	if err != nil {
		return editResponse{}, err
	}
	entryIndex, err := strconv.Atoi(request.Entry)
	if err != nil {
		return editResponse{}, ErrInvalidEntryIndex
	}
	renderEntry := func() (entryResponse, error) {
		return a.loadEntryPageByIndex(request.File, entryIndex)
	}
	return a.editAt(ctx, request.File, document, raw, entryIndex, editInput{
		Field: request.Field,
		Value: request.Value,
		View:  request.View,
	}, renderEntry)
}

func (a *handler) editUUID(ctx context.Context, request *editUUIDRequest) (editResponse, error) {
	raw, document, err := a.loadEditableDocument(request.File, request.Value)
	if err != nil {
		return editResponse{}, err
	}
	entryIndex, err := entryIndexByUUID(document, request.UUID)
	if errors.Is(err, cuebook.ErrBookEntryNotFound) {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: request.File}
	}
	if err != nil {
		return editResponse{}, err
	}
	renderEntry := func() (entryResponse, error) {
		return a.loadEntryPageByUUID(request.File, request.UUID)
	}
	return a.editAt(ctx, request.File, document, raw, entryIndex, editInput{
		Field: request.Field,
		Value: request.Value,
		View:  request.View,
	}, renderEntry)
}

func (a *handler) loadEditableDocument(fileName string, value *string) ([]byte, cuebook.Book, error) {
	if a.files == nil {
		return nil, cuebook.Book{}, ErrSourceReadOnly
	}
	if value == nil {
		return nil, cuebook.Book{}, ErrInvalidEditRequest
	}
	_, raw, document, err := a.loadEntryDocument(fileName)
	return raw, document, err
}

func (a *handler) editAt(ctx context.Context, fileName string, document cuebook.Book, raw []byte, entryIndex int, request editInput, renderEntry entryPageRenderer) (editResponse, error) {
	if entryIndex < 0 {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entryRef, err := document.GetEntryRef(entryIndex)
	if err != nil {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entry := entryRef.Entry()
	entryValue := entry.Value
	fieldName := request.Field
	field, ok := entry.GetFieldByName(fieldName)
	if !ok || fieldName == "" {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	if metadata.IsLockedField(field.Value) {
		return editResponse{}, FieldError{Code: ErrFieldLocked, Field: fieldName}
	}
	if isSecretField(field.Value) && *request.Value == "" && field.String() != "" {
		return a.renderEditResult(fileName, request.View, renderEntry)
	}

	value := *request.Value
	unset := value == "not_set" && field.Value.IncompleteKind() == cue.BoolKind && !isSecretField(field.Value)
	var change cuebook.Patch
	if unset {
		change, err = document.RemoveFieldValue(entryRef, fieldName)
	} else {
		change, err = document.UpdateFieldValue(entryRef, fieldName, value)
	}
	if err != nil {
		return editResponse{}, CauseError{Code: ErrFieldValueFormatFailed, Cause: err}
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return editResponse{}, CauseError{Code: ErrDocumentChangedBeforeEdit, Cause: err}
	}
	if unset {
		omitted := cuecontext.New().CompileBytes(candidate).LookupPath(cue.MakePath(cue.Index(entryIndex)))
		if !optionalField(omitted, fieldName) {
			return editResponse{}, FieldError{Code: ErrRequiredBooleanFieldUnset, Field: fieldName}
		}
		if !entryValue.LookupPath(cue.MakePath(cue.Str(fieldName))).Exists() {
			return a.renderEditResult(fileName, request.View, renderEntry)
		}
	}
	if _, err = cuebook.New(candidate); err != nil {
		return editResponse{}, CauseError{Code: ErrSubmittedValueConstraintViolation, Cause: err}
	}
	if err = a.commitFile(ctx, fileName, change); err != nil {
		return editResponse{}, err
	}
	return a.renderEditResult(fileName, request.View, renderEntry)
}

func (a *handler) renderEditResult(fileName, view string, renderEntry entryPageRenderer) (editResponse, error) {
	if view == "entry" {
		entry, err := renderEntry()
		if err != nil {
			return editResponse{}, err
		}
		return editResponse{
			pageData:    entry.pageData,
			destination: routeWithPrefix(a.routePrefix, entry.EntryURL),
		}, nil
	}
	data, err := a.renderEditedFile(fileName)
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
	UUID         string
	ShowEditIcon bool
	HideLabel    bool
	Markdown     template.HTML
}

func fieldEditURL(fileName string, entryIndex int, uuid, fieldName string, view, entryView bool) string {
	query := url.Values{}
	if uuid == "" {
		query.Set("entry", strconv.Itoa(entryIndex))
	} else {
		query.Set("uuid", uuid)
	}
	query.Set("field", fieldName)
	query.Set("file", fileName)
	if view {
		query.Set("mode", "view")
	}
	if entryView {
		query.Set("view", "entry")
	}
	editPath := "/edit"
	if uuid != "" {
		editPath += "/uuid"
	}
	return editPath + "?" + query.Encode()
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

func makeFieldView(field cuebook.Field, fileName string, index int, readOnly, entryView bool, entryUUID ...string) fieldView {
	uuid := ""
	if len(entryUUID) > 0 {
		uuid = entryUUID[0]
	}
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
		UUID:         uuid,
		Name:         field.Name,
		Description:  field.Description,
		Value:        value,
		Empty:        empty,
		EditURL:      fieldEditURL(fileName, index, uuid, field.Name, false, entryView),
		ViewURL:      fieldEditURL(fileName, index, uuid, field.Name, true, entryView),
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

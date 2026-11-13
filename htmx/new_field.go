package htmx

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
)

// newFieldRequest has a value per kind, because the form has a control per
// kind and submits all of them. Only the value of the chosen kind is used.
type newFieldRequest struct {
	File           string `schema:"file"`
	Entry          string `schema:"entry"`
	Name           string `schema:"name"`
	Kind           string `schema:"kind"`
	ValueString    string `schema:"value_string"`
	ValueMultiLine string `schema:"value_multiline"`
	ValueNumber    string `schema:"value_number"`
	ValueBoolean   string `schema:"value_boolean"`
}

func (r *newFieldRequest) value() string {
	return newFieldValue(r.Kind, r.ValueString, r.ValueMultiLine, r.ValueNumber, r.ValueBoolean)
}

func (*newFieldRequest) Validate(context.Context) error { return nil }

type newFieldUUIDRequest struct {
	File           string `schema:"file"`
	UUID           string `schema:"uuid"`
	Name           string `schema:"name"`
	Kind           string `schema:"kind"`
	ValueString    string `schema:"value_string"`
	ValueMultiLine string `schema:"value_multiline"`
	ValueNumber    string `schema:"value_number"`
	ValueBoolean   string `schema:"value_boolean"`
}

func (r *newFieldUUIDRequest) value() string {
	return newFieldValue(r.Kind, r.ValueString, r.ValueMultiLine, r.ValueNumber, r.ValueBoolean)
}

func (*newFieldUUIDRequest) Validate(context.Context) error { return nil }

func newFieldValue(kind, valueString, valueMultiLine, valueNumber, valueBoolean string) string {
	switch cuebook.FieldKind(kind) {
	case cuebook.FieldKindString:
		return valueString
	case cuebook.FieldKindMultiLine:
		return valueMultiLine
	case cuebook.FieldKindNumber:
		return valueNumber
	case cuebook.FieldKindBoolean:
		return valueBoolean
	default:
		return ""
	}
}

type newFieldKindView struct {
	Value   string
	Label   string
	Checked bool
}

func makeNewFieldKindViews() []newFieldKindView {
	kinds := cuebook.FieldKinds()
	views := make([]newFieldKindView, len(kinds))
	for index, kind := range kinds {
		views[index] = newFieldKindView{
			Value:   string(kind),
			Label:   kind.Label(),
			Checked: index == 0,
		}
	}
	return views
}

// newField adds a field of the chosen kind and value to an open entry. An
// empty value becomes the zero value of the kind.
func (a *handler) newField(ctx context.Context, request *newFieldRequest) (editResponse, error) {
	if a.files == nil {
		return editResponse{}, ErrSourceReadOnly
	}
	_, raw, document, err := a.loadEntryDocument(request.File)
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
	return a.addFieldAt(ctx, request.File, raw, document, entryIndex, request.Name, request.Kind, request.value(), renderEntry)
}

func (a *handler) newFieldUUID(ctx context.Context, request *newFieldUUIDRequest) (editResponse, error) {
	if a.files == nil {
		return editResponse{}, ErrSourceReadOnly
	}
	_, raw, document, err := a.loadEntryDocument(request.File)
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
	return a.addFieldAt(ctx, request.File, raw, document, entryIndex, request.Name, request.Kind, request.value(), renderEntry)
}

func (a *handler) addFieldAt(ctx context.Context, fileName string, raw []byte, document cuebook.Book, entryIndex int, fieldName, fieldKind, value string, renderEntry entryPageRenderer) (editResponse, error) {
	if entryIndex < 0 {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entryRef, err := document.GetEntryRef(entryIndex)
	if err != nil {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entry := entryRef.Entry()
	if !metadata.IsOpen(entry.Value) {
		return editResponse{}, ErrEntryDoesNotAcceptFields
	}

	name := strings.TrimSpace(fieldName)
	kind := cuebook.FieldKind(fieldKind)
	change, err := document.AddField(entryRef, name, kind, value)
	switch {
	case errors.Is(err, cuebook.ErrFieldNameEmpty):
		return editResponse{}, ErrFieldNameRequired
	case errors.Is(err, cuebook.ErrFieldExists):
		return editResponse{}, FieldError{Code: ErrFieldAlreadyExists, Field: name}
	case errors.Is(err, cuebook.ErrUnknownFieldKind):
		return editResponse{}, ErrFieldKindRequired
	case errors.Is(err, cuebook.ErrInvalidFieldValue) && kind == cuebook.FieldKindNumber:
		return editResponse{}, FieldError{Code: ErrFieldValueNotNumber, Field: name}
	case errors.Is(err, cuebook.ErrInvalidFieldValue):
		return editResponse{}, FieldError{Code: ErrFieldValueInvalid, Field: name}
	case err != nil:
		return editResponse{}, CauseError{Code: ErrUnableToAddField, Cause: err}
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return editResponse{}, CauseError{Code: ErrDocumentChangedBeforeFieldAdd, Cause: err}
	}
	if _, err = cuebook.New(candidate); err != nil {
		return editResponse{}, CauseError{Code: ErrNewFieldConstraintViolation, Cause: err}
	}
	if err = a.commitFile(ctx, fileName, change); err != nil {
		return editResponse{}, err
	}
	entryResponse, err := renderEntry()
	if err != nil {
		return editResponse{}, err
	}
	return editResponse{
		pageData:    entryResponse.pageData,
		destination: routeWithPrefix(a.routePrefix, entryResponse.EntryURL),
	}, nil
}

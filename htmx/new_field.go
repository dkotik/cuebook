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
	switch cuebook.FieldKind(r.Kind) {
	case cuebook.FieldKindString:
		return r.ValueString
	case cuebook.FieldKindMultiLine:
		return r.ValueMultiLine
	case cuebook.FieldKindNumber:
		return r.ValueNumber
	case cuebook.FieldKindBoolean:
		return r.ValueBoolean
	default:
		return ""
	}
}

func (*newFieldRequest) Validate(context.Context) error { return nil }

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
	entryRef, err := document.GetEntryRef(entryIndex)
	if err != nil {
		return editResponse{}, &cuebook.EntryNotFoundError{Path: fileName}
	}
	entry := entryRef.Entry()
	if !metadata.IsOpen(entry.Value) {
		return editResponse{}, ErrEntryDoesNotAcceptFields
	}

	// Surrounding spaces in a field name are almost always accidental.
	name := strings.TrimSpace(request.Name)
	kind := cuebook.FieldKind(request.Kind)
	change, err := document.AddField(entryRef, name, kind, request.value())
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
	return a.renderEditResponse(&editRequest{File: fileName, View: "entry"}, entryIndex)
}

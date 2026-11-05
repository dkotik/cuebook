package htmx

import (
	"context"
	"errors"
	"net/http"
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
		return editResponse{}, errors.New("This source is read-only.")
	}
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
	if !metadata.IsOpen(entryValue) {
		return editResponse{}, errors.New("This entry does not accept new fields.")
	}

	// Surrounding spaces in a field name are almost always accidental.
	name := strings.TrimSpace(request.Name)
	kind := cuebook.FieldKind(request.Kind)
	change, err := cuebook.AddField(raw, entryValue, name, kind, request.value())
	switch {
	case errors.Is(err, cuebook.ErrFieldNameEmpty):
		return editResponse{}, errors.New("The field name is required.")
	case errors.Is(err, cuebook.ErrFieldExists):
		return editResponse{}, errors.New("The field " + name + " already exists.")
	case errors.Is(err, cuebook.ErrUnknownFieldKind):
		return editResponse{}, errors.New("Choose a field type.")
	case errors.Is(err, cuebook.ErrInvalidFieldValue) && kind == cuebook.FieldKindNumber:
		return editResponse{}, errors.New("The value of field " + name + " is not a number.")
	case errors.Is(err, cuebook.ErrInvalidFieldValue):
		return editResponse{}, errors.New("The value of field " + name + " is invalid.")
	case err != nil:
		return editResponse{}, errors.New("The field could not be added to this entry.")
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return editResponse{}, errors.New("The entry changed before the field could be added. Reload and try again.")
	}
	if _, err = cuebook.New(candidate); err != nil {
		return editResponse{}, errors.New("The new field does not satisfy the CUE constraints: " + err.Error())
	}
	if err = a.commitFile(ctx, fileName, change); err != nil {
		if errors.Is(err, cuebook.ErrByteRangeNotFound) {
			return editResponse{}, errors.New("The entry changed before the field could be added. Reload and try again.")
		}
		return editResponse{}, errors.New("The field could not be saved.")
	}
	return a.renderEditResponse(&editRequest{File: fileName, View: "entry"}, entryIndex)
}

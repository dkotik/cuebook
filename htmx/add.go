package htmx

import (
	"context"
	"errors"
	"net/http"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/patch"
)

type addFieldRequest struct {
	Field string `schema:"field"`
	Value string `schema:"value"`
}

type addRequest struct {
	File    string            `schema:"file"`
	Entries []addFieldRequest `schema:"entry"`
}

func (*addRequest) Validate(context.Context) error { return nil }

type addResponse struct {
	pageData
}

func (a *handler) add(ctx context.Context, request *addRequest) (addResponse, error) {
	if a.committer == nil {
		return addResponseFrom(a.editFailure(ctx, "", "This source is read-only.", http.StatusForbidden))
	}

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return addResponseFrom(a.editFailure(ctx, "", "Unable to list CUE files.", http.StatusInternalServerError))
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return addResponseFrom(a.editFailure(ctx, fileName, message, status))
	}
	page, _ := a.pageForDocument(fileName, fileNames, raw, document, "")
	fieldNames := make([]string, len(request.Entries))
	values := make([]string, len(request.Entries))
	for index, entry := range request.Entries {
		fieldNames[index] = entry.Field
		values[index] = entry.Value
	}

	definitions := entryFieldDefinitions(document)
	if len(definitions) == 0 {
		return addResponseFrom(withNotice(page, "This document has no entry fields to add."), http.StatusUnprocessableEntity)
	}
	if len(fieldNames) == 0 || len(fieldNames) != len(values) {
		return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The add-entry fields are invalid.", http.StatusBadRequest))
	}

	knownFields := make(map[string]entryFieldDefinition, len(definitions))
	for _, definition := range definitions {
		knownFields[definition.Field.Name] = definition
	}
	submitted := make(map[string]string, len(fieldNames))
	for i, name := range fieldNames {
		if _, ok := knownFields[name]; !ok {
			return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The add-entry form contains an unknown field.", http.StatusBadRequest))
		}
		if _, exists := submitted[name]; exists {
			return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The add-entry form contains a duplicate field.", http.StatusBadRequest))
		}
		submitted[name] = values[i]
	}

	declarations := make([]any, 0, len(definitions))
	for _, definition := range definitions {
		field := definition.Field
		value, ok := submitted[field.Name]
		if !ok {
			if definition.Optional {
				continue
			}
			return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "A required entry field is missing.", http.StatusBadRequest))
		}
		if definition.Optional && value == "" {
			continue
		}
		declaration, err := addFieldDeclaration(field, value)
		if err != nil {
			return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The value for field "+field.Name+" is invalid.", http.StatusUnprocessableEntity))
		}
		declarations = append(declarations, declaration)
	}

	entry := cuecontext.New().BuildExpr(ast.NewStruct(declarations...))
	if err := entry.Err(); err != nil {
		return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The submitted entry is invalid.", http.StatusUnprocessableEntity))
	}
	change, err := patch.AppendToStructList(raw, entry)
	if err != nil {
		return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The entry could not be added to this CUE document.", http.StatusUnprocessableEntity))
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The document changed before the entry could be added. Reload and try again.", http.StatusConflict))
	}
	if _, err := cuebook.New(candidate); err != nil {
		return addResponseFrom(a.renderAddFormError(page, fieldNames, values, "The submitted entry does not satisfy the CUE constraints: "+err.Error(), http.StatusUnprocessableEntity))
	}
	if err := a.commitFile(fileName, change); err != nil {
		status := http.StatusInternalServerError
		notice := "The entry could not be saved."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The document changed before the entry could be added. Reload and try again."
		}
		return addResponseFrom(a.renderAddFormError(page, fieldNames, values, notice, status))
	}
	return addResponseFrom(a.finishEdit(fileName))
}

func (a *handler) renderAddFormError(page pageData, fieldNames, values []string, message string, status int) (pageData, int) {
	page.AddFields = retainAddFieldValues(page.AddFields, fieldNames, values)
	page.AddError = message
	return page, status
}

func retainAddFieldValues(fields []addFieldView, names, values []string) []addFieldView {
	valuesByName := make(map[string]string, min(len(names), len(values)))
	for index := 0; index < min(len(names), len(values)); index++ {
		if _, exists := valuesByName[names[index]]; !exists {
			valuesByName[names[index]] = values[index]
		}
	}
	fields = append([]addFieldView(nil), fields...)
	for index := range fields {
		if fields[index].Secret {
			continue
		}
		if value, ok := valuesByName[fields[index].Name]; ok {
			fields[index].Value = value
		}
	}
	return fields
}

func addFieldDeclaration(field cuebook.Field, input string) (ast.Decl, error) {
	if field.Value.IncompleteKind()&cue.StringKind != 0 {
		return field.WithValue(input)
	}

	value, err := metadata.FormatAccordingToAttributes(field.Value, input)
	if err != nil {
		return nil, err
	}
	compiled := cuecontext.New().CompileString(value)
	if err := compiled.Err(); err != nil {
		return nil, err
	}
	expression, ok := compiled.Syntax().(ast.Expr)
	if !ok {
		return nil, errors.New("field value is not a CUE expression")
	}
	return &ast.Field{
		Label: ast.NewString(field.Name),
		Value: expression,
	}, nil
}

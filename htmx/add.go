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

func (a *handler) add(_ context.Context, request *addRequest) (addResponse, error) {
	if a.committer == nil {
		return addResponse{}, errors.New("This source is read-only.")
	}

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return addResponse{}, errors.New("Unable to list CUE files.")
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		return addResponse{}, errors.New(message)
	}
	fieldNames := make([]string, len(request.Entries))
	values := make([]string, len(request.Entries))
	for index, entry := range request.Entries {
		fieldNames[index] = entry.Field
		values[index] = entry.Value
	}

	definitions := entryFieldDefinitions(document)
	if len(definitions) == 0 {
		return addResponse{}, errors.New("This document has no entry fields to add.")
	}
	if len(fieldNames) == 0 || len(fieldNames) != len(values) {
		return addResponse{}, errors.New("The add-entry fields are invalid.")
	}

	knownFields := make(map[string]entryFieldDefinition, len(definitions))
	for _, definition := range definitions {
		knownFields[definition.Field.Name] = definition
	}
	submitted := make(map[string]string, len(fieldNames))
	for i, name := range fieldNames {
		if _, ok := knownFields[name]; !ok {
			return addResponse{}, errors.New("The add-entry form contains an unknown field.")
		}
		if _, exists := submitted[name]; exists {
			return addResponse{}, errors.New("The add-entry form contains a duplicate field.")
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
			return addResponse{}, errors.New("A required entry field is missing.")
		}
		if definition.Optional && value == "" {
			continue
		}
		declaration, err := addFieldDeclaration(field, value)
		if err != nil {
			return addResponse{}, errors.New("The value for field " + field.Name + " is invalid.")
		}
		declarations = append(declarations, declaration)
	}

	entry := cuecontext.New().BuildExpr(ast.NewStruct(declarations...))
	if err := entry.Err(); err != nil {
		return addResponse{}, errors.New("The submitted entry is invalid.")
	}
	change, err := patch.AppendToStructList(raw, entry)
	if err != nil {
		return addResponse{}, errors.New("The entry could not be added to this CUE document.")
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return addResponse{}, errors.New("The document changed before the entry could be added. Reload and try again.")
	}
	if _, err := cuebook.New(candidate); err != nil {
		return addResponse{}, errors.New("The submitted entry does not satisfy the CUE constraints: " + err.Error())
	}
	if err := a.commitFile(fileName, change); err != nil {
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			return addResponse{}, errors.New("The document changed before the entry could be added. Reload and try again.")
		}
		return addResponse{}, errors.New("The entry could not be saved.")
	}
	data, err := a.renderEditedFile(fileName)
	return addResponse{pageData: data}, err
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

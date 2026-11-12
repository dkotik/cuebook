package htmx

import (
	"context"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
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
	if a.files == nil {
		return addResponse{}, ErrSourceReadOnly
	}

	fileName := request.File
	fileNames, err := a.fileNames()
	if err != nil {
		return addResponse{}, ErrUnableToListCueFiles
	}
	raw, document, err := a.readDocument(fileName, fileNames)
	if err != nil {
		return addResponse{}, err
	}
	fieldNames := make([]string, len(request.Entries))
	values := make([]string, len(request.Entries))
	for index, entry := range request.Entries {
		fieldNames[index] = entry.Field
		values[index] = entry.Value
	}

	definitions := entryFieldDefinitions(document)
	if len(definitions) == 0 {
		return addResponse{}, ErrNoEntryFieldsToAdd
	}
	if len(fieldNames) == 0 || len(fieldNames) != len(values) {
		return addResponse{}, ErrInvalidAddEntryFields
	}

	knownFields := make(map[string]entryFieldDefinition, len(definitions))
	for _, definition := range definitions {
		knownFields[definition.Field.Name] = definition
	}
	submitted := make(map[string]string, len(fieldNames))
	for i, name := range fieldNames {
		if _, ok := knownFields[name]; !ok {
			return addResponse{}, ErrUnknownAddEntryField
		}
		if _, exists := submitted[name]; exists {
			return addResponse{}, ErrDuplicateAddEntryField
		}
		submitted[name] = values[i]
	}

	declarations := make([]any, 0, len(definitions))
	for _, definition := range definitions {
		field := definition.Field
		optional := definition.Optional
		boolean := field.Value.IncompleteKind() == cue.BoolKind && !isSecretField(field.Value)

		value, ok := submitted[field.Name]
		if !ok {
			if optional {
				continue
			}
			return addResponse{}, ErrRequiredEntryFieldMissing
		}
		if value == "not_set" && boolean {
			if !optional {
				return addResponse{}, FieldError{Code: ErrRequiredBooleanFieldUnset, Field: field.Name}
			}
			continue
		}
		if optional && value == "" {
			continue
		}
		declaration, err := addFieldDeclaration(field, value)
		if err != nil {
			return addResponse{}, FieldError{Code: ErrAddFieldValueInvalid, Field: field.Name}
		}
		declarations = append(declarations, declaration)
	}

	entry := cuecontext.New().BuildExpr(ast.NewStruct(declarations...))
	if err := entry.Err(); err != nil {
		return addResponse{}, CauseError{Code: ErrSubmittedEntryInvalid, Cause: err}
	}
	entryIndex, err := document.Len()
	if err != nil {
		return addResponse{}, CauseError{Code: ErrUnableToDetermineEntryPosition, Cause: err}
	}
	if err := a.appendEntry(ctx, fileName, raw, entry); err != nil {
		return addResponse{}, err
	}
	addedEntry, err := a.loadEntryPage(fileName, cuebook.ByteRange{}, entryIndex)
	if err != nil {
		return addResponse{}, ErrEntrySavedButDisplayFailed
	}
	return addResponse{pageData: addedEntry.pageData}, nil
}

func (a *handler) appendEntry(ctx context.Context, fileName string, raw []byte, entry cue.Value) error {
	book, err := cuebook.New(raw)
	if err != nil {
		return CauseError{Code: ErrUnableToAddEntry, Cause: err}
	}
	change, err := book.AppendToStructList(entry)
	if err != nil {
		return CauseError{Code: ErrUnableToAddEntry, Cause: err}
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		return CauseError{Code: ErrDocumentChangedBeforeEntryAdd, Cause: err}
	}
	if _, err := cuebook.New(candidate); err != nil {
		return CauseError{Code: ErrSubmittedEntryConstraintViolation, Cause: err}
	}
	if err := a.commitFile(ctx, fileName, change); err != nil {
		return err
	}
	return nil
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
		return nil, ErrFieldValueNotCUEExpression
	}
	return &ast.Field{
		Label: ast.NewString(field.Name),
		Value: expression,
	}, nil
}

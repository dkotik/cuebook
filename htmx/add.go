package htmx

import (
	"errors"
	"net/http"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/cuecontext"
	"github.com/dkotik/cuebook"
	"github.com/dkotik/cuebook/metadata"
	"github.com/dkotik/cuebook/patch"
)

func (a *handler) add(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		a.renderPage(w, r, pageData{ReadOnly: a.committer == nil, Error: "Cross-origin edits are not allowed."}, http.StatusForbidden)
		return
	}
	if a.committer == nil {
		a.editFailure(w, r, "", "This source is read-only.", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		a.editFailure(w, r, "", "The add-entry request is invalid.", http.StatusBadRequest)
		return
	}

	fileName := r.PostForm.Get("file")
	fileNames, err := a.fileNames()
	if err != nil {
		a.editFailure(w, r, "", "Unable to list CUE files.", http.StatusInternalServerError)
		return
	}
	raw, document, status, message := a.readDocument(fileName, fileNames)
	if status != http.StatusOK {
		a.editFailure(w, r, fileName, message, status)
		return
	}
	page, _ := a.pageForDocument(fileName, fileNames, raw, document, "")
	fieldNames, fieldsPresent := r.PostForm["field"]
	values, valuesPresent := r.PostForm["value"]

	definitions := entryFieldDefinitions(document)
	if len(definitions) == 0 {
		a.renderPage(w, r, withNotice(page, "This document has no entry fields to add."), http.StatusUnprocessableEntity)
		return
	}
	if !fieldsPresent || !valuesPresent || len(fieldNames) == 0 || len(fieldNames) != len(values) {
		a.renderAddFormError(w, r, page, fieldNames, values, "The add-entry fields are invalid.", http.StatusBadRequest)
		return
	}

	knownFields := make(map[string]entryFieldDefinition, len(definitions))
	for _, definition := range definitions {
		knownFields[definition.Field.Name] = definition
	}
	submitted := make(map[string]string, len(fieldNames))
	for i, name := range fieldNames {
		if _, ok := knownFields[name]; !ok {
			a.renderAddFormError(w, r, page, fieldNames, values, "The add-entry form contains an unknown field.", http.StatusBadRequest)
			return
		}
		if _, exists := submitted[name]; exists {
			a.renderAddFormError(w, r, page, fieldNames, values, "The add-entry form contains a duplicate field.", http.StatusBadRequest)
			return
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
			a.renderAddFormError(w, r, page, fieldNames, values, "A required entry field is missing.", http.StatusBadRequest)
			return
		}
		if definition.Optional && value == "" {
			continue
		}
		declaration, err := addFieldDeclaration(field, value)
		if err != nil {
			a.renderAddFormError(w, r, page, fieldNames, values, "The value for field "+field.Name+" is invalid.", http.StatusUnprocessableEntity)
			return
		}
		declarations = append(declarations, declaration)
	}

	entry := cuecontext.New().BuildExpr(ast.NewStruct(declarations...))
	if err := entry.Err(); err != nil {
		a.renderAddFormError(w, r, page, fieldNames, values, "The submitted entry is invalid.", http.StatusUnprocessableEntity)
		return
	}
	change, err := patch.AppendToStructList(raw, entry)
	if err != nil {
		a.renderAddFormError(w, r, page, fieldNames, values, "The entry could not be added to this CUE document.", http.StatusUnprocessableEntity)
		return
	}
	candidate, err := change.ApplyToCueSource(raw)
	if err != nil {
		a.renderAddFormError(w, r, page, fieldNames, values, "The document changed before the entry could be added. Reload and try again.", http.StatusConflict)
		return
	}
	if _, err := cuebook.New(candidate); err != nil {
		a.renderAddFormError(w, r, page, fieldNames, values, "The submitted entry does not satisfy the CUE constraints: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err := a.commitFile(fileName, change); err != nil {
		status := http.StatusInternalServerError
		notice := "The entry could not be saved."
		if errors.Is(err, patch.ErrByteRangeNotFound) {
			status = http.StatusConflict
			notice = "The document changed before the entry could be added. Reload and try again."
		}
		a.renderAddFormError(w, r, page, fieldNames, values, notice, status)
		return
	}
	a.finishEdit(w, r, fileName)
}

func (a *handler) renderAddFormError(w http.ResponseWriter, r *http.Request, page pageData, fieldNames, values []string, message string, status int) {
	page.AddFields = retainAddFieldValues(page.AddFields, fieldNames, values)
	page.AddError = message
	a.renderPage(w, r, page, status)
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

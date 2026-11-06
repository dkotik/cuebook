package htmx

import (
	"fmt"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Error identifies an error raised while serving an HTMX request.
type Error uint8

const (
	// ErrUnknown is the zero value for an unspecified HTMX error.
	ErrUnknown Error = iota
	// ErrSourceReadOnly occurs when a write is requested without a writable filesystem.
	ErrSourceReadOnly
	// ErrUnableToListCueFiles occurs when the source filesystem cannot be listed.
	ErrUnableToListCueFiles
	// ErrNoEntryFieldsToAdd occurs when a document has no schema fields for new entries.
	ErrNoEntryFieldsToAdd
	// ErrInvalidAddEntryFields occurs when add-form field and value counts are inconsistent.
	ErrInvalidAddEntryFields
	// ErrUnknownAddEntryField occurs when an add form submits an undefined field.
	ErrUnknownAddEntryField
	// ErrDuplicateAddEntryField occurs when an add form submits a field more than once.
	ErrDuplicateAddEntryField
	// ErrRequiredEntryFieldMissing occurs when an add form omits a required field.
	ErrRequiredEntryFieldMissing
	// ErrRequiredBooleanFieldUnset occurs when a required boolean field is unset.
	ErrRequiredBooleanFieldUnset
	// ErrAddFieldValueInvalid occurs when an add form contains a value that cannot be formatted.
	ErrAddFieldValueInvalid
	// ErrSubmittedEntryInvalid occurs when the constructed entry is invalid.
	ErrSubmittedEntryInvalid
	// ErrUnableToDetermineEntryPosition occurs when the destination index cannot be determined.
	ErrUnableToDetermineEntryPosition
	// ErrEntrySavedButDisplayFailed occurs when a saved entry cannot be rendered afterward.
	ErrEntrySavedButDisplayFailed
	// ErrEditSavedButDisplayFailed occurs when a saved edit cannot be rendered afterward.
	ErrEditSavedButDisplayFailed
	// ErrUnableToAddEntry occurs when an entry cannot be appended to a CUE document.
	ErrUnableToAddEntry
	// ErrDocumentChangedBeforeEntryAdd occurs when the source changes before an entry is added.
	ErrDocumentChangedBeforeEntryAdd
	// ErrDocumentChangedBeforeEdit occurs when the source changes before an edit is applied.
	ErrDocumentChangedBeforeEdit
	// ErrDocumentChangedBeforeFieldAdd occurs when the source changes before a field is added.
	ErrDocumentChangedBeforeFieldAdd
	// ErrSubmittedEntryConstraintViolation occurs when a submitted entry violates CUE constraints.
	ErrSubmittedEntryConstraintViolation
	// ErrFieldValueNotCUEExpression occurs when a field value cannot be represented as a CUE expression.
	ErrFieldValueNotCUEExpression
	// ErrEntryAlreadyArchived occurs when an entry in the archive is archived again.
	ErrEntryAlreadyArchived
	// ErrInvalidEntryPosition occurs when an entry position is not a valid integer.
	ErrInvalidEntryPosition
	// ErrUnableToReadEntryCount occurs when the document's entry count cannot be read.
	ErrUnableToReadEntryCount
	// ErrUnableToCreateArchiveFile occurs when the archive file cannot be created.
	ErrUnableToCreateArchiveFile
	// ErrInvalidEntryIndex occurs when an entry index cannot be parsed.
	ErrInvalidEntryIndex
	// ErrInvalidEditRequest occurs when an edit request omits its value.
	ErrInvalidEditRequest
	// ErrFieldLocked occurs when an edit targets a locked field.
	ErrFieldLocked
	// ErrFieldValueFormatFailed occurs when a submitted field value cannot be formatted.
	ErrFieldValueFormatFailed
	// ErrSubmittedValueConstraintViolation occurs when an edited value violates CUE constraints.
	ErrSubmittedValueConstraintViolation
	// ErrInvalidEntryRequest occurs when an entry request contains invalid byte-range data.
	ErrInvalidEntryRequest
	// ErrUnableToDisplayCUEDocument occurs when the CUE document cannot be displayed.
	ErrUnableToDisplayCUEDocument
	// ErrUnableToLocateEntry occurs when an entry cannot be located in its source document.
	ErrUnableToLocateEntry
	// ErrGitChangesViewNotConfigured occurs when the Git changes view is requested without a linker.
	ErrGitChangesViewNotConfigured
	// ErrUnableToLoadGitChangesView occurs when the Git changes view cannot be loaded.
	ErrUnableToLoadGitChangesView
	// ErrFileSystemNil occurs when NewWithFileSystem receives no filesystem.
	ErrFileSystemNil
	// ErrSourceFileSystemNil occurs when handler construction receives no source filesystem.
	ErrSourceFileSystemNil
	// ErrUnableToReorderEntries occurs when the entries cannot be reordered.
	ErrUnableToReorderEntries
	// ErrReorderedDocumentConstraintViolation occurs when the reordered document violates CUE constraints.
	ErrReorderedDocumentConstraintViolation
	// ErrUnableToPrepareEntryTransfer occurs when an entry transfer cannot be prepared.
	ErrUnableToPrepareEntryTransfer
	// ErrDestinationConstraintViolation occurs when a transferred entry violates destination constraints.
	ErrDestinationConstraintViolation
	// ErrDestinationChangedBeforeTransfer occurs when the destination changes during a transfer.
	ErrDestinationChangedBeforeTransfer
	// ErrSourceChangedBeforeTransfer occurs when the source changes during a transfer.
	ErrSourceChangedBeforeTransfer
	// ErrSourceConstraintViolation occurs when removing a transferred entry violates source constraints.
	ErrSourceConstraintViolation
	// ErrEntryDoesNotAcceptFields occurs when a closed entry is asked to accept a new field.
	ErrEntryDoesNotAcceptFields
	// ErrFieldNameRequired occurs when a new field is submitted without a name.
	ErrFieldNameRequired
	// ErrFieldAlreadyExists occurs when a new field duplicates an existing field.
	ErrFieldAlreadyExists
	// ErrFieldKindRequired occurs when a new field request has no recognized kind.
	ErrFieldKindRequired
	// ErrFieldValueNotNumber occurs when a numeric field receives a non-number value.
	ErrFieldValueNotNumber
	// ErrFieldValueInvalid occurs when a new field value is invalid.
	ErrFieldValueInvalid
	// ErrUnableToAddField occurs when a new field cannot be added to an entry.
	ErrUnableToAddField
	// ErrNewFieldConstraintViolation occurs when a new field violates CUE constraints.
	ErrNewFieldConstraintViolation
	// ErrPastedSourceEmpty occurs when the pasted source contains only whitespace.
	ErrPastedSourceEmpty
	// ErrUnableToParsePastedCUEStruct occurs when pasted CUE cannot be parsed as a struct.
	ErrUnableToParsePastedCUEStruct
	// ErrIncompleteCutInformation occurs when a cut request omits required identifying data.
	ErrIncompleteCutInformation
	// ErrInvalidCutEntryPosition occurs when the cut entry position cannot be parsed.
	ErrInvalidCutEntryPosition
	// ErrCutEntryChangedOrMoved occurs when a cut entry no longer matches its saved fingerprint.
	ErrCutEntryChangedOrMoved
	// ErrSearchIndexBuilding occurs when a search is requested before indexing completes.
	ErrSearchIndexBuilding
	// ErrUnableToBuildSearchIndex occurs when the search index reports a build failure.
	ErrUnableToBuildSearchIndex
	// ErrSearchQueryRequired occurs when a search request has an empty query.
	ErrSearchQueryRequired
	// ErrSearchQueryInvalid occurs when a search query cannot be parsed.
	ErrSearchQueryInvalid
	// ErrAgentHandlerNil occurs when configuring a nil agent handler.
	ErrAgentHandlerNil
	// ErrAgentAlreadyConfigured occurs when an agent handler is configured more than once.
	ErrAgentAlreadyConfigured
	// ErrGitFileLinkerNil occurs when configuring a nil Git file linker.
	ErrGitFileLinkerNil
	// ErrGitChangesAlreadyConfigured occurs when Git changes are configured more than once.
	ErrGitChangesAlreadyConfigured
	// ErrAdaptorNil occurs when configuring a nil adaptor.
	ErrAdaptorNil
	// ErrAdaptorAlreadyConfigured occurs when an adaptor is configured more than once.
	ErrAdaptorAlreadyConfigured
	// ErrServeMuxNil occurs when configuring a nil HTTP serve mux.
	ErrServeMuxNil
	// ErrServeMuxAlreadyConfigured occurs when a serve mux is configured more than once.
	ErrServeMuxAlreadyConfigured
)

var errorMessages = map[Error]i18n.Message{
	ErrSourceReadOnly:                       {ID: "htmx.sourceReadOnly", Other: "This source is read-only."},
	ErrUnableToListCueFiles:                 {ID: "htmx.unableToListCueFiles", Other: "Unable to list CUE files."},
	ErrNoEntryFieldsToAdd:                   {ID: "htmx.noEntryFieldsToAdd", Other: "This document has no entry fields to add."},
	ErrInvalidAddEntryFields:                {ID: "htmx.invalidAddEntryFields", Other: "The add-entry fields are invalid."},
	ErrUnknownAddEntryField:                 {ID: "htmx.unknownAddEntryField", Other: "The add-entry form contains an unknown field."},
	ErrDuplicateAddEntryField:               {ID: "htmx.duplicateAddEntryField", Other: "The add-entry form contains a duplicate field."},
	ErrRequiredEntryFieldMissing:            {ID: "htmx.requiredEntryFieldMissing", Other: "A required entry field is missing."},
	ErrRequiredBooleanFieldUnset:            {ID: "htmx.requiredBooleanFieldUnset", Other: "A required boolean field cannot be unset."},
	ErrAddFieldValueInvalid:                 {ID: "htmx.addFieldValueInvalid", Other: "The field value is invalid."},
	ErrSubmittedEntryInvalid:                {ID: "htmx.submittedEntryInvalid", Other: "The submitted entry is invalid."},
	ErrUnableToDetermineEntryPosition:       {ID: "htmx.unableToDetermineEntryPosition", Other: "Unable to determine where to add the entry."},
	ErrEntrySavedButDisplayFailed:           {ID: "htmx.entrySavedButDisplayFailed", Other: "The entry was saved, but the updated document could not be displayed."},
	ErrEditSavedButDisplayFailed:            {ID: "htmx.editSavedButDisplayFailed", Other: "The edit was saved, but the updated document could not be displayed."},
	ErrUnableToAddEntry:                     {ID: "htmx.unableToAddEntry", Other: "The entry could not be added to this CUE document."},
	ErrDocumentChangedBeforeEntryAdd:        {ID: "htmx.documentChangedBeforeEntryAdd", Other: "The document changed before the entry could be added. Reload and try again."},
	ErrDocumentChangedBeforeEdit:            {ID: "htmx.documentChangedBeforeEdit", Other: "The entry changed before the edit could be applied. Reload and try again."},
	ErrDocumentChangedBeforeFieldAdd:        {ID: "htmx.documentChangedBeforeFieldAdd", Other: "The entry changed before the field could be added. Reload and try again."},
	ErrSubmittedEntryConstraintViolation:    {ID: "htmx.submittedEntryConstraintViolation", Other: "The submitted entry does not satisfy the CUE constraints"},
	ErrFieldValueNotCUEExpression:           {ID: "htmx.fieldValueNotCUEExpression", Other: "field value is not a CUE expression"},
	ErrEntryAlreadyArchived:                 {ID: "htmx.entryAlreadyArchived", Other: "Entries in the archive cannot be archived again."},
	ErrInvalidEntryPosition:                 {ID: "htmx.invalidEntryPosition", Other: "The entry position is invalid."},
	ErrUnableToReadEntryCount:               {ID: "htmx.unableToReadEntryCount", Other: "Unable to read the entry count."},
	ErrUnableToCreateArchiveFile:            {ID: "htmx.unableToCreateArchiveFile", Other: "Unable to create the archive file."},
	ErrInvalidEntryIndex:                    {ID: "htmx.invalidEntryIndex", Other: "The entry index is invalid."},
	ErrInvalidEditRequest:                   {ID: "htmx.invalidEditRequest", Other: "The edit request is invalid."},
	ErrFieldLocked:                          {ID: "htmx.fieldLocked", Other: "The field is locked and cannot be changed."},
	ErrFieldValueFormatFailed:               {ID: "htmx.fieldValueFormatFailed", Other: "The field value could not be formatted."},
	ErrSubmittedValueConstraintViolation:    {ID: "htmx.submittedValueConstraintViolation", Other: "The submitted value does not satisfy the CUE constraints"},
	ErrInvalidEntryRequest:                  {ID: "htmx.invalidEntryRequest", Other: "The entry request is invalid."},
	ErrUnableToDisplayCUEDocument:           {ID: "htmx.unableToDisplayCUEDocument", Other: "Unable to display this CUE document."},
	ErrUnableToLocateEntry:                  {ID: "htmx.unableToLocateEntry", Other: "Unable to locate this entry in the CUE file."},
	ErrGitChangesViewNotConfigured:          {ID: "htmx.gitChangesViewNotConfigured", Other: "The Git changes view is not configured."},
	ErrUnableToLoadGitChangesView:           {ID: "htmx.unableToLoadGitChangesView", Other: "Unable to load the Git changes view."},
	ErrFileSystemNil:                        {ID: "htmx.fileSystemNil", Other: "htmx: file system is nil"},
	ErrSourceFileSystemNil:                  {ID: "htmx.sourceFileSystemNil", Other: "htmx: source filesystem is nil"},
	ErrUnableToReorderEntries:               {ID: "htmx.unableToReorderEntries", Other: "The entries could not be reordered."},
	ErrReorderedDocumentConstraintViolation: {ID: "htmx.reorderedDocumentConstraintViolation", Other: "The reordered document does not satisfy the CUE constraints."},
	ErrUnableToPrepareEntryTransfer:         {ID: "htmx.unableToPrepareEntryTransfer", Other: "The entry transfer could not be prepared."},
	ErrDestinationConstraintViolation:       {ID: "htmx.destinationConstraintViolation", Other: "The entry does not satisfy the destination file's CUE constraints."},
	ErrDestinationChangedBeforeTransfer:     {ID: "htmx.destinationChangedBeforeTransfer", Other: "The destination file changed before the entry could be added. Reload and try again."},
	ErrSourceChangedBeforeTransfer:          {ID: "htmx.sourceChangedBeforeTransfer", Other: "The source file changed before the entry could be removed. Reload and try again."},
	ErrSourceConstraintViolation:            {ID: "htmx.sourceConstraintViolation", Other: "Removing the entry does not satisfy the source file's CUE constraints."},
	ErrEntryDoesNotAcceptFields:             {ID: "htmx.entryDoesNotAcceptFields", Other: "This entry does not accept new fields."},
	ErrFieldNameRequired:                    {ID: "htmx.fieldNameRequired", Other: "The field name is required."},
	ErrFieldAlreadyExists:                   {ID: "htmx.fieldAlreadyExists", Other: "The field already exists."},
	ErrFieldKindRequired:                    {ID: "htmx.fieldKindRequired", Other: "Choose a field type."},
	ErrFieldValueNotNumber:                  {ID: "htmx.fieldValueNotNumber", Other: "The field value is not a number."},
	ErrFieldValueInvalid:                    {ID: "htmx.fieldValueInvalid", Other: "The field value is invalid."},
	ErrUnableToAddField:                     {ID: "htmx.unableToAddField", Other: "The field could not be added to this entry."},
	ErrNewFieldConstraintViolation:          {ID: "htmx.newFieldConstraintViolation", Other: "The new field does not satisfy the CUE constraints"},
	ErrPastedSourceEmpty:                    {ID: "htmx.pastedSourceEmpty", Other: "The pasted source is empty."},
	ErrUnableToParsePastedCUEStruct:         {ID: "htmx.unableToParsePastedCUEStruct", Other: "Unable to parse the pasted CUE struct"},
	ErrIncompleteCutInformation:             {ID: "htmx.incompleteCutInformation", Other: "The cut information is incomplete. Copy or cut the entry again."},
	ErrInvalidCutEntryPosition:              {ID: "htmx.invalidCutEntryPosition", Other: "The cut entry position is invalid. Cut the entry again."},
	ErrCutEntryChangedOrMoved:               {ID: "htmx.cutEntryChangedOrMoved", Other: "The cut entry changed or moved. Copy or cut the entry again."},
	ErrSearchIndexBuilding:                  {ID: "htmx.searchIndexBuilding", Other: "The search index is still being built."},
	ErrUnableToBuildSearchIndex:             {ID: "htmx.unableToBuildSearchIndex", Other: "Unable to build the search index."},
	ErrSearchQueryRequired:                  {ID: "htmx.searchQueryRequired", Other: "A search query is required."},
	ErrSearchQueryInvalid:                   {ID: "htmx.searchQueryInvalid", Other: "The search query is invalid."},
	ErrAgentHandlerNil:                      {ID: "htmx.agentHandlerNil", Other: "htmx: agent handler is nil"},
	ErrAgentAlreadyConfigured:               {ID: "htmx.agentAlreadyConfigured", Other: "htmx: agent handler is already set"},
	ErrGitFileLinkerNil:                     {ID: "htmx.gitFileLinkerNil", Other: "htmx: Git file linker is nil"},
	ErrGitChangesAlreadyConfigured:          {ID: "htmx.gitChangesAlreadyConfigured", Other: "htmx: Git changes view is already configured"},
	ErrAdaptorNil:                           {ID: "htmx.adaptorNil", Other: "htmx: adaptor is nil"},
	ErrAdaptorAlreadyConfigured:             {ID: "htmx.adaptorAlreadyConfigured", Other: "htmx: adaptor is already set"},
	ErrServeMuxNil:                          {ID: "htmx.serveMuxNil", Other: "htmx: serve mux is nil"},
	ErrServeMuxAlreadyConfigured:            {ID: "htmx.serveMuxAlreadyConfigured", Other: "htmx: serve mux is already set"},
}

func (e Error) Error() string {
	if message, ok := errorMessages[e]; ok {
		return message.Other
	}
	return fmt.Sprintf("unknown HTMX error code #%d", e)
}

func (e Error) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message, ok := errorMessages[e]
	if !ok {
		message = i18n.Message{ID: "htmx.unknownError", Other: "unknown HTMX error code #{{.Code}}"}
	}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]any{"Code": uint8(e)},
	})
}

// FieldError carries a field name for an HTMX field-validation error.
type FieldError struct {
	Code  Error
	Field string
}

func (e FieldError) Error() string {
	switch e.Code {
	case ErrRequiredBooleanFieldUnset:
		return fmt.Sprintf("The required boolean field %s cannot be unset.", e.Field)
	case ErrAddFieldValueInvalid:
		return fmt.Sprintf("The value for field %s is invalid.", e.Field)
	case ErrFieldLocked:
		return fmt.Sprintf("The field %s is locked and cannot be changed.", e.Field)
	case ErrFieldAlreadyExists:
		return fmt.Sprintf("The field %s already exists.", e.Field)
	case ErrFieldValueNotNumber:
		return fmt.Sprintf("The value of field %s is not a number.", e.Field)
	case ErrFieldValueInvalid:
		return fmt.Sprintf("The value of field %s is invalid.", e.Field)
	default:
		return e.Code.Error()
	}
}

func (e FieldError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := errorMessages[e.Code]
	switch e.Code {
	case ErrRequiredBooleanFieldUnset:
		message.Other = "The required boolean field {{.Field}} cannot be unset."
	case ErrAddFieldValueInvalid:
		message.Other = "The value for field {{.Field}} is invalid."
	case ErrFieldLocked:
		message.Other = "The field {{.Field}} is locked and cannot be changed."
	case ErrFieldAlreadyExists:
		message.Other = "The field {{.Field}} already exists."
	case ErrFieldValueNotNumber:
		message.Other = "The value of field {{.Field}} is not a number."
	case ErrFieldValueInvalid:
		message.Other = "The value of field {{.Field}} is invalid."
	}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]any{"Field": e.Field},
	})
}

// CauseError preserves a cause while providing HTMX-specific context.
type CauseError struct {
	Code  Error
	Cause error
}

func (e CauseError) Error() string {
	if e.Cause == nil {
		return e.Code.Error()
	}
	return e.Code.Error() + ": " + e.Cause.Error()
}

func (e CauseError) Unwrap() error {
	return e.Cause
}

func (e CauseError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	prefix, err := e.Code.LocalizeWith(localizer)
	if err != nil || e.Cause == nil {
		return prefix, err
	}
	cause := e.Cause.Error()
	if localizable, ok := e.Cause.(interface {
		LocalizeWith(*i18n.Localizer) (string, error)
	}); ok {
		cause, err = localizable.LocalizeWith(localizer)
		if err != nil {
			return "", err
		}
	}
	return prefix + ": " + cause, nil
}

// MessageError carries a dynamic HTMX message that does not have a fixed error code.
type MessageError struct {
	Message string
}

func (e MessageError) Error() string {
	return e.Message
}

func (e MessageError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	message := i18n.Message{ID: "htmx.dynamicError", Other: e.Message}
	return localizer.Localize(&i18n.LocalizeConfig{DefaultMessage: &message})
}

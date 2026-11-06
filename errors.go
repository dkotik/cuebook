package cuebook

import (
	"fmt"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Error identifies an error raised while parsing or accessing a Cue book.
type Error uint8

const (
	// ErrUnknownCuebookError is the zero value for an unspecified cuebook error.
	ErrUnknownCuebookError Error = iota
	_                            // Retains subsequent Error values after removing the parse code.
	_                            // Retains subsequent Error values after removing the validation code.
	// ErrNotCueList occurs when the root CUE value is not a list.
	ErrNotCueList
	_ // Retains subsequent Error values after removing unused list iteration code.
	// ErrCannotLoadAbstractEntry occurs when an abstract or null list item is loaded as an entry.
	ErrCannotLoadAbstractEntry
	// ErrEntryValueNotStruct occurs when a list item is not a structured object.
	ErrEntryValueNotStruct
	_ // Retains subsequent Error values after removing the field-iteration code.
	// ErrBookEntryIndexOutOfRange occurs when a requested entry index is invalid.
	ErrBookEntryIndexOutOfRange
	_ // Retains subsequent Error values after removing the unused value lookup code.
	_ // Retains subsequent Error values after removing the unused list-length code.
	// ErrFieldIndexOutOfRange occurs when a field index is negative or exceeds an entry's fields.
	ErrFieldIndexOutOfRange
)

func (e Error) Error() string {
	switch e {
	case ErrNotCueList:
		return "expected a list structued data"
	case ErrCannotLoadAbstractEntry:
		return "cannot load an abstract value as structured object"
	case ErrEntryValueNotStruct:
		return "value is not a structured object"
	case ErrBookEntryIndexOutOfRange, ErrFieldIndexOutOfRange:
		return "not found: index out of range"
	default:
		return fmt.Sprintf("unknown cuebook error code #%d", e)
	}
}

func (e Error) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	var message i18n.Message
	switch e {
	case ErrNotCueList:
		message = i18n.Message{ID: "cuebook.expectedList", Other: "expected a list structued data"}
	case ErrCannotLoadAbstractEntry:
		message = i18n.Message{ID: "cuebook.abstractEntry", Other: "cannot load an abstract value as structured object"}
	case ErrEntryValueNotStruct:
		message = i18n.Message{ID: "cuebook.entryValueNotStruct", Other: "value is not a structured object"}
	case ErrBookEntryIndexOutOfRange:
		message = i18n.Message{ID: "cuebook.entryIndexOutOfRange", Other: "not found: index out of range"}
	case ErrFieldIndexOutOfRange:
		message = i18n.Message{ID: "entry.fieldIndexOutOfRange", Other: "not found: index out of range"}
	default:
		message = i18n.Message{ID: "cuebook.unknownError", Other: "unknown cuebook error code #{{.Code}}"}
	}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]any{"Code": uint8(e)},
	})
}

// PatchError identifies a source-change error code.
type PatchError uint8

const (
	ErrUnknown PatchError = iota
	ErrSourceIsNotList
	ErrByteRangesOverlap
	// ErrByteRangeNotFound occurs when a byte range cannot be located.
	ErrByteRangeNotFound
	ErrEntryNotStruct
	ErrTargetFieldNotStructField
	ErrDoubleCommaAtListEnd
	ErrNotListOfStructs
	ErrRootListEndNotFound
	ErrFieldNameEmpty
	ErrFieldExists
	ErrUnknownFieldKind
	ErrInvalidFieldValue
)

// Localizable represents errors that can return localized messages.
type Localizable interface {
	LocalizeWith(*i18n.Localizer) (string, error)
}

func (e PatchError) Error() string {
	switch e {
	case ErrSourceIsNotList:
		return "source is not a list"
	case ErrByteRangesOverlap:
		return "byte ranges overlap"
	case ErrByteRangeNotFound:
		return "byte range not found"
	case ErrEntryNotStruct:
		return "entry not a struct"
	case ErrTargetFieldNotStructField:
		return "target field not a struct field"
	case ErrDoubleCommaAtListEnd:
		return "double comma at list end"
	case ErrNotListOfStructs:
		return "not a list of structs"
	case ErrRootListEndNotFound:
		return "root list end not found"
	case ErrFieldNameEmpty:
		return "field name is empty"
	case ErrFieldExists:
		return "field already exists"
	case ErrUnknownFieldKind:
		return "unknown field kind"
	case ErrInvalidFieldValue:
		return "field value does not match its kind"
	default:
		return fmt.Sprintf("unknown patch error code #%d", e)
	}
}

func (e PatchError) LocalizeWith(localizer *i18n.Localizer) (string, error) {
	var message i18n.Message
	switch e {
	case ErrSourceIsNotList:
		message = i18n.Message{ID: "patch.sourceIsNotList", Other: "source is not a list"}
	case ErrByteRangesOverlap:
		message = i18n.Message{ID: "patch.byteRangesOverlap", Other: "byte ranges overlap"}
	case ErrByteRangeNotFound:
		message = i18n.Message{ID: "patch.byteRangeNotFound", Other: "byte range not found"}
	case ErrEntryNotStruct:
		message = i18n.Message{ID: "patch.entryNotStruct", Other: "entry not a struct"}
	case ErrTargetFieldNotStructField:
		message = i18n.Message{ID: "patch.targetFieldNotStructField", Other: "target field not a struct field"}
	case ErrDoubleCommaAtListEnd:
		message = i18n.Message{ID: "patch.doubleCommaAtListEnd", Other: "double comma at list end"}
	case ErrNotListOfStructs:
		message = i18n.Message{ID: "patch.notListOfStructs", Other: "not a list of structs"}
	case ErrRootListEndNotFound:
		message = i18n.Message{ID: "patch.rootListEndNotFound", Other: "root list end not found"}
	case ErrFieldNameEmpty:
		message = i18n.Message{ID: "patch.fieldNameEmpty", Other: "field name is empty"}
	case ErrFieldExists:
		message = i18n.Message{ID: "patch.fieldExists", Other: "field already exists"}
	case ErrUnknownFieldKind:
		message = i18n.Message{ID: "patch.unknownFieldKind", Other: "unknown field kind"}
	case ErrInvalidFieldValue:
		message = i18n.Message{ID: "patch.invalidFieldValue", Other: "field value does not match its kind"}
	default:
		message = i18n.Message{ID: "patch.unknownError", Other: "unknown patch error code #{{.Code}}"}
	}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]interface{}{"Code": uint8(e)},
	})
}

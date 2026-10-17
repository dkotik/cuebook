package cuebook

import (
	"fmt"
	"net/http"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// PatchError identifies a source-change error code.
type PatchError uint8

const (
	ErrUnknown PatchError = iota
	ErrSourceIsNotList
	ErrByteRangesOverlap
	ErrUnableToFormatFieldValue
	ErrEntryNotStruct
	ErrTargetFieldNotStructField
	ErrDoubleCommaAtListEnd
	ErrNotListOfStructs
	ErrRootListEndNotFound
)

// FileNotFoundError reports that a requested CUE file does not exist.
type FileNotFoundError struct {
	Path string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("file not found: path=%s", e.Path)
}

func (e *FileNotFoundError) HyperTextStatusCode() int {
	return http.StatusNotFound
}

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
	case ErrUnableToFormatFieldValue:
		return "failed to format field value"
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
	case ErrUnableToFormatFieldValue:
		message = i18n.Message{ID: "patch.formatError", Other: "failed to format field value"}
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
	default:
		message = i18n.Message{ID: "patch.unknownError", Other: "unknown patch error code #{{.Code}}"}
	}
	return localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &message,
		TemplateData:   map[string]interface{}{"Code": uint8(e)},
	})
}

var _ Localizable = PatchError(0)

package patch

import (
	"fmt"

	"github.com/dkotik/cuebook"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Error identifies a patch error code.
type Error uint8

const (
	ErrUnknown Error = iota
	ErrSourceIsNotList
	ErrByteRangesOverlap
	ErrUnableToFormatFieldValue
	ErrEntryNotStruct
	ErrTargetFieldNotStructField
	ErrDoubleCommaAtListEnd
	ErrNotListOfStructs
	ErrRootListEndNotFound
)

// ErrByteRangeNotFound aliases the root package's byte-range lookup error.
var ErrByteRangeNotFound = cuebook.ErrByteRangeNotFound

// Localizable represents errors that can return localized messages.
type Localizable interface {
	LocalizeWith(*i18n.Localizer) (string, error)
}

func (e Error) Error() string {
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

func (e Error) LocalizeWith(localizer *i18n.Localizer) (string, error) {
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

var _ Localizable = Error(0)

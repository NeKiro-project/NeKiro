// Package configcenter defines provider-neutral configuration source semantics.
package configcenter

const (
	maxKeyLength        = 160
	maxKeySegmentLength = 32
	maxProviderIDLength = 128
)

// Key identifies one opaque configuration value. Its string form is strict and
// is never cleaned, trimmed, or case-folded.
type Key struct {
	value string
}

// ParseKey validates a strict slash-separated configuration key.
func ParseKey(value string) (Key, error) {
	if !validKey(value) {
		return Key{}, NewError(CodeInvalid, ErrorDetails{Operation: OperationValidateKey})
	}
	return Key{value: value}, nil
}

// String returns the exact validated key text.
func (key Key) String() string {
	return key.value
}

// Valid reports whether key was constructed from a valid strict key.
func (key Key) Valid() bool {
	return validKey(key.value)
}

// ProviderID selects one explicitly composed configuration reader. It is
// validated by ParseProviderID and by Factory construction/lookup.
type ProviderID string

// ParseProviderID validates an exact provider identifier.
func ParseProviderID(value string) (ProviderID, error) {
	provider := ProviderID(value)
	if !provider.Valid() {
		return "", NewError(CodeInvalid, ErrorDetails{Operation: OperationValidateProvider})
	}
	return provider, nil
}

// String returns the exact provider identifier.
func (provider ProviderID) String() string {
	return string(provider)
}

// Valid reports whether provider is a strict provider identifier.
func (provider ProviderID) Valid() bool {
	value := string(provider)
	if len(value) == 0 || len(value) > maxProviderIDLength {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		if index == 0 {
			if !isASCIIAlphaNumeric(character) {
				return false
			}
			continue
		}
		if !isASCIIAlphaNumeric(character) && character != '.' && character != '_' && character != ':' && character != '-' {
			return false
		}
	}
	return true
}

func validKey(value string) bool {
	if len(value) == 0 || len(value) > maxKeyLength {
		return false
	}
	segmentLength := 0
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character == '/' {
			if segmentLength == 0 {
				return false
			}
			segmentLength = 0
			continue
		}
		if segmentLength == 0 {
			if !isASCIIAlphaNumeric(character) {
				return false
			}
		} else if !isASCIIAlphaNumeric(character) && character != '.' && character != '_' && character != ':' && character != '-' {
			return false
		}
		segmentLength++
		if segmentLength > maxKeySegmentLength {
			return false
		}
	}
	return segmentLength != 0
}

func isASCIIAlphaNumeric(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}

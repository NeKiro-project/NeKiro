package file

import (
	"encoding/base64"
	"strings"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

const (
	mappingPrefix = "cfg-v1-"
	mappingSuffix = ".value"
	// MaxMappedLeafLength is the maximum direct-child filename for a valid key:
	// 7-byte prefix + RawURL(160-byte key) + 6-byte suffix.
	MaxMappedLeafLength = 7 + 214 + 6
)

// MapKey maps one strict key to a flat, separator-free root child name.
func MapKey(key configcenter.Key) (string, error) {
	if !key.Valid() {
		return "", configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Key:       key,
			Operation: configcenter.OperationObserve,
		})
	}
	encoded := base64.RawURLEncoding.EncodeToString([]byte(key.String()))
	leaf := mappingPrefix + encoded + mappingSuffix
	if len(leaf) > MaxMappedLeafLength {
		return "", configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Key:       key,
			Operation: configcenter.OperationObserve,
		})
	}
	return leaf, nil
}

// UnmapLeaf reverses MapKey and verifies canonical unpadded Base64url. It
// rejects unrelated or malformed filenames rather than normalizing them.
func UnmapLeaf(leaf string) (configcenter.Key, error) {
	if !strings.HasPrefix(leaf, mappingPrefix) || !strings.HasSuffix(leaf, mappingSuffix) {
		return configcenter.Key{}, configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Operation: configcenter.OperationObserve,
		})
	}
	encoded := strings.TrimSuffix(strings.TrimPrefix(leaf, mappingPrefix), mappingSuffix)
	if encoded == "" {
		return configcenter.Key{}, configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Operation: configcenter.OperationObserve,
		})
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != encoded {
		return configcenter.Key{}, configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Operation: configcenter.OperationObserve,
		})
	}
	key, err := configcenter.ParseKey(string(decoded))
	if err != nil {
		return configcenter.Key{}, err
	}
	return key, nil
}

func mappedKeyFromEventName(name string) (configcenter.Key, bool) {
	leaf := name
	if strings.ContainsAny(leaf, `/\\`) {
		return configcenter.Key{}, false
	}
	if !strings.HasPrefix(leaf, mappingPrefix) || !strings.HasSuffix(leaf, mappingSuffix) {
		return configcenter.Key{}, false
	}
	key, err := UnmapLeaf(leaf)
	if err != nil {
		return configcenter.Key{}, false
	}
	return key, true
}

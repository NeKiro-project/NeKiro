package configcenter

import "reflect"

// Factory is an immutable, caller-composed provider-to-reader map. It has no
// registration side effect, source precedence, publisher selection, or default
// provider.
type Factory struct {
	readers map[ProviderID]DynamicConfiguration
}

// NewFactory copies an exact provider-to-reader map. A nil map is rejected as
// absent composition; an empty, non-nil map is valid and simply has no known
// providers.
func NewFactory(readers map[ProviderID]DynamicConfiguration) (*Factory, error) {
	if readers == nil {
		return nil, NewError(CodeInvalid, ErrorDetails{Operation: OperationFactory})
	}

	copyReaders := make(map[ProviderID]DynamicConfiguration, len(readers))
	for provider, reader := range readers {
		if !provider.Valid() || nilDynamicConfiguration(reader) {
			return nil, NewError(CodeInvalid, ErrorDetails{
				Provider:  provider,
				Operation: OperationFactory,
			})
		}
		copyReaders[provider] = reader
	}
	return &Factory{readers: copyReaders}, nil
}

// Reader resolves exactly the provider selected by the caller.
func (factory *Factory) Reader(provider ProviderID) (DynamicConfiguration, error) {
	if factory == nil || !provider.Valid() {
		return nil, NewError(CodeInvalid, ErrorDetails{
			Provider:  provider,
			Operation: OperationFactory,
		})
	}
	reader, ok := factory.readers[provider]
	if !ok {
		return nil, NewError(CodeUnsupported, ErrorDetails{
			Provider:  provider,
			Operation: OperationFactory,
		})
	}
	return reader, nil
}

func nilDynamicConfiguration(reader DynamicConfiguration) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

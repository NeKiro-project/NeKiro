package stream

import (
	"encoding/json"
	"errors"

	"github.com/Nene7ko/NeKiro/contracts"
)

var ErrInterrupted = errors.New("A2A stream ended before a terminal event")

// Event is the transport-neutral representation of one validated upstream
// A2A stream event. Payload is transient caller data and must never be written
// to the Invocation Ledger.
type Event struct {
	Payload        json.RawMessage
	Kind           string
	TaskID         string
	ContextID      string
	ArtifactID     string
	ArtifactAppend bool
	ArtifactLast   bool
	TerminalType   contracts.ResultStreamEventType
	TerminalStatus string
	ErrorCode      contracts.PlatformErrorCode
}

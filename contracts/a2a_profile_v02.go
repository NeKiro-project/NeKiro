package contracts

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/a2aproject/a2a-go/a2a"
)

const (
	A2AProfileSchemaVersionV02 = "0.2"
	A2AProfileProtocolVersion  = "0.3.0"
	A2AProfileSDKModule        = "github.com/a2aproject/a2a-go"
	A2AProfileSDKVersion       = "v0.3.15"
	A2AConformanceSchemaV01    = "0.1"
)

const a2aConformanceCorpusRoot = "a2a-profile/v0.3.0/conformance"

//go:embed schemas/a2a-profile.v0.2.schema.json a2a-profile/v0.3.0/profile.v0.2.json a2a-profile/v0.3.0/conformance/*.json a2a-profile/v0.3.0/conformance/*.sse
var a2aProfileV02Files embed.FS

type A2AProfileV02 struct {
	SchemaVersion   string                   `json:"schemaVersion"`
	ProtocolVersion string                   `json:"protocolVersion"`
	SDK             A2ASDK                   `json:"sdk"`
	Transport       string                   `json:"transport"`
	AgentCardPath   string                   `json:"agentCardPath"`
	Operations      []A2AProfileOperationV02 `json:"operations"`
	TaskStates      A2AProfileTaskStatesV02  `json:"taskStates"`
	ContextHeaders  A2AContextHeaders        `json:"contextHeaders"`
	Conformance     A2AProfileConformanceV02 `json:"conformance"`
}

type A2AProfileOperationV02 struct {
	Method              string   `json:"method"`
	ClientMethod        string   `json:"clientMethod"`
	ServerMethod        string   `json:"serverMethod"`
	Interaction         string   `json:"interaction"`
	RequestType         string   `json:"requestType"`
	AcceptedResultKinds []string `json:"acceptedResultKinds,omitempty"`
	AcceptedEventKinds  []string `json:"acceptedEventKinds,omitempty"`
	ExpectedErrors      []string `json:"expectedErrors,omitempty"`
}

type A2AProfileTaskStatesV02 struct {
	Transient   []A2AProfileTaskStateV02 `json:"transient"`
	Terminal    []A2AProfileTaskStateV02 `json:"terminal"`
	Unsupported []A2AProfileTaskStateV02 `json:"unsupported"`
	Unspecified A2AProfileTaskStateV02   `json:"unspecified"`
}

type A2AProfileTaskStateV02 struct {
	State            string            `json:"state"`
	InvocationStatus string            `json:"invocationStatus"`
	ErrorCode        PlatformErrorCode `json:"errorCode,omitempty"`
}

type A2AProfileConformanceV02 struct {
	Manifest         string `json:"manifest"`
	FixtureAuthority string `json:"fixtureAuthority"`
	JSONRPCVersion   string `json:"jsonrpcVersion"`
	SSEMediaType     string `json:"sseMediaType"`
}

type A2AConformanceManifestV02 struct {
	SchemaVersion        string                  `json:"schemaVersion"`
	ProfileSchemaVersion string                  `json:"profileSchemaVersion"`
	ProtocolVersion      string                  `json:"protocolVersion"`
	Cases                []A2AConformanceCaseV02 `json:"cases"`
}

type A2AConformanceCaseV02 struct {
	ID             string                    `json:"id"`
	File           string                    `json:"file"`
	RequestFile    string                    `json:"requestFile,omitempty"`
	Operation      string                    `json:"operation"`
	FixtureKind    string                    `json:"fixtureKind"`
	ExpectedValid  bool                      `json:"expectedValid"`
	WireResultKind string                    `json:"wireResultKind,omitempty"`
	GoConcreteType string                    `json:"goConcreteType,omitempty"`
	ProtocolError  A2AProtocolErrorV02       `json:"protocolError,omitempty"`
	MediaType      string                    `json:"mediaType"`
	Rules          []A2AConformanceRuleIDV02 `json:"rules"`
}

type A2AConformanceRuleIDV02 string

const (
	A2ARuleJSONRPCEnvelope         A2AConformanceRuleIDV02 = "jsonrpc-envelope"
	A2ARuleRequestParams           A2AConformanceRuleIDV02 = "request-params"
	A2ARuleRequestResponseID       A2AConformanceRuleIDV02 = "request-response-id"
	A2ARuleResultXORError          A2AConformanceRuleIDV02 = "result-xor-error"
	A2ARuleResultUnion             A2AConformanceRuleIDV02 = "result-union"
	A2ARuleResultType              A2AConformanceRuleIDV02 = "result-type"
	A2ARuleMessageResult           A2AConformanceRuleIDV02 = "message-result"
	A2ARuleTaskIdentity            A2AConformanceRuleIDV02 = "task-identity"
	A2ARuleTaskState               A2AConformanceRuleIDV02 = "task-state"
	A2ARuleSSEFraming              A2AConformanceRuleIDV02 = "sse-framing"
	A2ARuleEventKinds              A2AConformanceRuleIDV02 = "event-kinds"
	A2ARuleTaskContextStability    A2AConformanceRuleIDV02 = "task-context-stability"
	A2ARuleTerminalRequired        A2AConformanceRuleIDV02 = "terminal-required"
	A2ARuleTerminalLast            A2AConformanceRuleIDV02 = "terminal-last"
	A2ARuleArtifactOrder           A2AConformanceRuleIDV02 = "artifact-order"
	A2ARuleArtifactLastChunk       A2AConformanceRuleIDV02 = "artifact-last-chunk"
	A2ARuleHistoryLength           A2AConformanceRuleIDV02 = "history-length"
	A2ARuleErrorOnly               A2AConformanceRuleIDV02 = "error-only"
	A2ARuleRejectedMapping         A2AConformanceRuleIDV02 = "rejected-mapping"
	A2ARuleUnsupportedStateMapping A2AConformanceRuleIDV02 = "unsupported-state-mapping"
	A2ARuleSameTask                A2AConformanceRuleIDV02 = "same-task"
	A2ARuleCanceledState           A2AConformanceRuleIDV02 = "canceled-state"
	A2ARuleFiveContextHeaders      A2AConformanceRuleIDV02 = "five-context-headers"
)

var a2aConformanceRuleExecutionOrderV02 = []A2AConformanceRuleIDV02{
	A2ARuleSSEFraming,
	A2ARuleJSONRPCEnvelope,
	A2ARuleResultXORError,
	A2ARuleRequestResponseID,
	A2ARuleRequestParams,
	A2ARuleResultUnion,
	A2ARuleResultType,
	A2ARuleMessageResult,
	A2ARuleTaskIdentity,
	A2ARuleTaskState,
	A2ARuleEventKinds,
	A2ARuleTaskContextStability,
	A2ARuleTerminalRequired,
	A2ARuleTerminalLast,
	A2ARuleArtifactOrder,
	A2ARuleArtifactLastChunk,
	A2ARuleHistoryLength,
	A2ARuleErrorOnly,
	A2ARuleRejectedMapping,
	A2ARuleUnsupportedStateMapping,
	A2ARuleSameTask,
	A2ARuleCanceledState,
	A2ARuleFiveContextHeaders,
}

type A2AProtocolErrorV02 string

const (
	A2AProtocolErrorInvalidJSONRPCEnvelope    A2AProtocolErrorV02 = "invalid-jsonrpc-envelope"
	A2AProtocolErrorInvalidJSONRPCVersion     A2AProtocolErrorV02 = "invalid-jsonrpc-version"
	A2AProtocolErrorInvalidResponseIDType     A2AProtocolErrorV02 = "invalid-response-id-type"
	A2AProtocolErrorResponseIDMismatch        A2AProtocolErrorV02 = "response-id-mismatch"
	A2AProtocolErrorResultErrorExclusivity    A2AProtocolErrorV02 = "result-error-exclusivity"
	A2AProtocolErrorResultErrorRequired       A2AProtocolErrorV02 = "result-error-required"
	A2AProtocolErrorInvalidResultKind         A2AProtocolErrorV02 = "invalid-result-kind"
	A2AProtocolErrorInvalidMessageResult      A2AProtocolErrorV02 = "invalid-message-result"
	A2AProtocolErrorInvalidTask               A2AProtocolErrorV02 = "invalid-task"
	A2AProtocolErrorUnsupportedTaskState      A2AProtocolErrorV02 = "unsupported-task-state"
	A2AProtocolErrorTaskContextMismatch       A2AProtocolErrorV02 = "task-context-mismatch"
	A2AProtocolErrorEventAfterTerminal        A2AProtocolErrorV02 = "event-after-terminal"
	A2AProtocolErrorEOFWithoutTerminal        A2AProtocolErrorV02 = "eof-without-terminal"
	A2AProtocolErrorArtifactAppendWithoutBase A2AProtocolErrorV02 = "artifact-append-without-base"
	A2AProtocolErrorArtifactAfterLastChunk    A2AProtocolErrorV02 = "artifact-after-last-chunk"
	A2AProtocolErrorTaskNotFound              A2AProtocolErrorV02 = "task-not-found"
	A2AProtocolErrorTaskNotCancelable         A2AProtocolErrorV02 = "task-not-cancelable"
	A2AProtocolErrorAssertionPrerequisite     A2AProtocolErrorV02 = "assertion-prerequisite-failed"
)

type A2AConformanceAssertionErrorV02 struct {
	Rule           A2AConformanceRuleIDV02
	Classification A2AProtocolErrorV02
	Cause          error
}

func (e *A2AConformanceAssertionErrorV02) Error() string {
	return fmt.Sprintf("A2A conformance rule %q classified failure as %q: %v", e.Rule, e.Classification, e.Cause)
}

func (e *A2AConformanceAssertionErrorV02) Unwrap() error {
	return e.Cause
}

type a2aJSONRPCResponseV02 struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

func newA2AConformanceAssertionErrorV02(rule A2AConformanceRuleIDV02, classification A2AProtocolErrorV02, cause error) error {
	return &A2AConformanceAssertionErrorV02{
		Rule:           rule,
		Classification: classification,
		Cause:          cause,
	}
}

func validateA2AJSONRPCResponseEnvelopeV02(data []byte) error {
	response, err := decodeA2AJSONRPCResponseV02(data)
	if err != nil {
		return err
	}
	if response.JSONRPC != "2.0" {
		return newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidJSONRPCVersion,
			fmt.Errorf("JSON-RPC version = %q, want 2.0", response.JSONRPC),
		)
	}
	if len(response.ID) == 0 {
		return newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidResponseIDType,
			fmt.Errorf("JSON-RPC response id is missing"),
		)
	}
	if err := validateA2AJSONRPCResponseIDV02(response.ID); err != nil {
		return newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidResponseIDType,
			err,
		)
	}
	return nil
}

func validateA2AJSONRPCResponseBaselineV02(data []byte) error {
	if err := validateA2AJSONRPCResponseEnvelopeV02(data); err != nil {
		return err
	}
	return validateA2AJSONRPCResultXORErrorV02(data)
}

func validateA2AJSONRPCResultXORErrorV02(data []byte) error {
	response, err := decodeA2AJSONRPCResponseV02(data)
	if err != nil {
		return err
	}
	hasResult := len(response.Result) > 0
	hasError := len(response.Error) > 0
	switch {
	case hasResult && hasError:
		return newA2AConformanceAssertionErrorV02(
			A2ARuleResultXORError,
			A2AProtocolErrorResultErrorExclusivity,
			fmt.Errorf("JSON-RPC response contains both result and error"),
		)
	case !hasResult && !hasError:
		return newA2AConformanceAssertionErrorV02(
			A2ARuleResultXORError,
			A2AProtocolErrorResultErrorRequired,
			fmt.Errorf("JSON-RPC response contains neither result nor error"),
		)
	default:
		return nil
	}
}

func decodeA2AJSONRPCResponseV02(data []byte) (a2aJSONRPCResponseV02, error) {
	if err := rejectDuplicateJSONMemberNames(data); err != nil {
		return a2aJSONRPCResponseV02{}, newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidJSONRPCEnvelope,
			fmt.Errorf("decode JSON-RPC response: %w", err),
		)
	}

	var object map[string]json.RawMessage
	objectDecoder := json.NewDecoder(bytes.NewReader(data))
	if err := objectDecoder.Decode(&object); err != nil {
		return a2aJSONRPCResponseV02{}, newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidJSONRPCEnvelope,
			fmt.Errorf("decode JSON-RPC response object: %w", err),
		)
	}
	if object == nil {
		return a2aJSONRPCResponseV02{}, newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidJSONRPCEnvelope,
			fmt.Errorf("decode JSON-RPC response object: top-level value is not an object"),
		)
	}
	if err := requireJSONEOF(objectDecoder); err != nil {
		return a2aJSONRPCResponseV02{}, newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidJSONRPCEnvelope,
			fmt.Errorf("decode JSON-RPC response object: %w", err),
		)
	}

	var response a2aJSONRPCResponseV02
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return a2aJSONRPCResponseV02{}, newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidJSONRPCEnvelope,
			fmt.Errorf("decode JSON-RPC response: %w", err),
		)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return a2aJSONRPCResponseV02{}, newA2AConformanceAssertionErrorV02(
			A2ARuleJSONRPCEnvelope,
			A2AProtocolErrorInvalidJSONRPCEnvelope,
			fmt.Errorf("decode JSON-RPC response: %w", err),
		)
	}
	return response, nil
}

func validateA2AJSONRPCResponseIDV02(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("decode JSON-RPC response id: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return fmt.Errorf("decode JSON-RPC response id: %w", err)
	}
	switch value.(type) {
	case nil, string, json.Number:
		return nil
	default:
		return fmt.Errorf("JSON-RPC response id has unsupported JSON type %T", value)
	}
}

func validateA2AConformanceCaseOutcomeV02(testCase A2AConformanceCaseV02, actual error) error {
	caseLabel := fmt.Sprintf("A2A conformance case %q", testCase.ID)
	if testCase.ExpectedValid {
		if actual != nil {
			return fmt.Errorf("%s expected valid: %w", caseLabel, actual)
		}
		return nil
	}
	if actual == nil {
		return fmt.Errorf("%s declared protocolError %q but no assertion failed", caseLabel, testCase.ProtocolError)
	}

	var assertionError *A2AConformanceAssertionErrorV02
	if !errors.As(actual, &assertionError) {
		return fmt.Errorf("%s produced an unclassified assertion failure: %w", caseLabel, actual)
	}
	if assertionError.Classification != testCase.ProtocolError {
		return fmt.Errorf(
			"%s actual protocol failure %q does not match declared protocolError %q: %w",
			caseLabel,
			assertionError.Classification,
			testCase.ProtocolError,
			actual,
		)
	}
	if !a2aProtocolErrorHasClaimedRule(
		assertionError.Classification,
		map[A2AConformanceRuleIDV02]struct{}{assertionError.Rule: {}},
	) {
		return fmt.Errorf(
			"%s actual rule %q cannot establish protocol failure %q",
			caseLabel,
			assertionError.Rule,
			assertionError.Classification,
		)
	}
	for _, ruleID := range testCase.Rules {
		if ruleID == assertionError.Rule {
			return nil
		}
	}
	return fmt.Errorf("%s actual failure rule %q was not declared", caseLabel, assertionError.Rule)
}

type a2aManifestField[T any] struct {
	Value   T
	Present bool
	Null    bool
}

func (f *a2aManifestField[T]) UnmarshalJSON(data []byte) error {
	f.Present = true
	if bytes.Equal(data, []byte("null")) {
		f.Null = true
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&f.Value); err != nil {
		return err
	}
	return requireJSONEOF(decoder)
}

type a2aConformanceManifestJSON struct {
	SchemaVersion        a2aManifestField[string]                   `json:"schemaVersion"`
	ProfileSchemaVersion a2aManifestField[string]                   `json:"profileSchemaVersion"`
	ProtocolVersion      a2aManifestField[string]                   `json:"protocolVersion"`
	Cases                a2aManifestField[[]a2aConformanceCaseJSON] `json:"cases"`
}

type a2aConformanceCaseJSON struct {
	ID             a2aManifestField[string]                    `json:"id"`
	File           a2aManifestField[string]                    `json:"file"`
	RequestFile    a2aManifestField[string]                    `json:"requestFile"`
	Operation      a2aManifestField[string]                    `json:"operation"`
	FixtureKind    a2aManifestField[string]                    `json:"fixtureKind"`
	ExpectedValid  a2aManifestField[bool]                      `json:"expectedValid"`
	WireResultKind a2aManifestField[string]                    `json:"wireResultKind"`
	GoConcreteType a2aManifestField[string]                    `json:"goConcreteType"`
	ProtocolError  a2aManifestField[A2AProtocolErrorV02]       `json:"protocolError"`
	MediaType      a2aManifestField[string]                    `json:"mediaType"`
	Rules          a2aManifestField[[]A2AConformanceRuleIDV02] `json:"rules"`
}

type A2ATaskStateClassification string

const (
	A2ATaskStateTransient A2ATaskStateClassification = "transient"
	A2ATaskStateTerminal  A2ATaskStateClassification = "terminal"
)

type A2ATaskStateMapping struct {
	State            a2a.TaskState
	Classification   A2ATaskStateClassification
	InvocationStatus string
	ErrorCode        PlatformErrorCode
}

type A2AProfileStateError struct {
	State     a2a.TaskState
	Reason    string
	ErrorCode PlatformErrorCode
}

func (e *A2AProfileStateError) Error() string {
	return fmt.Sprintf("A2A task state %q is %s in Profile v0.2", e.State, e.Reason)
}

type A2AProfileTaskError struct {
	Reason    string
	ErrorCode PlatformErrorCode
}

func (e *A2AProfileTaskError) Error() string {
	return fmt.Sprintf("A2A task violates Profile v0.2: %s", e.Reason)
}

type A2AProfileMessageError struct {
	Reason    string
	ErrorCode PlatformErrorCode
}

func (e *A2AProfileMessageError) Error() string {
	return fmt.Sprintf("A2A message violates Profile v0.2: %s", e.Reason)
}

func LoadA2AProfileV02() (A2AProfileV02, error) {
	return decodeA2AProfileV02[A2AProfileV02]("a2a-profile/v0.3.0/profile.v0.2.json")
}

func LoadA2AConformanceManifestV02() (A2AConformanceManifestV02, error) {
	corpus, err := fs.Sub(a2aProfileV02Files, a2aConformanceCorpusRoot)
	if err != nil {
		return A2AConformanceManifestV02{}, fmt.Errorf("open A2A conformance corpus: %w", err)
	}
	data, err := readA2AConformanceFixtureV02(corpus, "manifest.json")
	if err != nil {
		return A2AConformanceManifestV02{}, err
	}
	return DecodeA2AConformanceManifestV02(data, corpus)
}

func A2AProfileV02Files() fs.FS {
	return a2aProfileV02Files
}

func ReadA2AConformanceFixtureV02(fixturePath string) ([]byte, error) {
	corpus, err := fs.Sub(a2aProfileV02Files, a2aConformanceCorpusRoot)
	if err != nil {
		return nil, fmt.Errorf("open A2A conformance corpus: %w", err)
	}
	return readA2AConformanceFixtureV02(corpus, fixturePath)
}

func DecodeA2AConformanceManifestV02(data []byte, corpus fs.FS) (A2AConformanceManifestV02, error) {
	if corpus == nil {
		return A2AConformanceManifestV02{}, fmt.Errorf("decode A2A conformance manifest: corpus is missing")
	}
	if err := rejectDuplicateJSONMemberNames(data); err != nil {
		return A2AConformanceManifestV02{}, fmt.Errorf("decode A2A conformance manifest: %w", err)
	}

	var document a2aConformanceManifestJSON
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return A2AConformanceManifestV02{}, fmt.Errorf("decode A2A conformance manifest: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return A2AConformanceManifestV02{}, fmt.Errorf("decode A2A conformance manifest: %w", err)
	}

	schemaVersion, err := requireA2AManifestField("schemaVersion", document.SchemaVersion)
	if err != nil {
		return A2AConformanceManifestV02{}, err
	}
	profileSchemaVersion, err := requireA2AManifestField("profileSchemaVersion", document.ProfileSchemaVersion)
	if err != nil {
		return A2AConformanceManifestV02{}, err
	}
	protocolVersion, err := requireA2AManifestField("protocolVersion", document.ProtocolVersion)
	if err != nil {
		return A2AConformanceManifestV02{}, err
	}
	wireCases, err := requireA2AManifestField("cases", document.Cases)
	if err != nil {
		return A2AConformanceManifestV02{}, err
	}
	if schemaVersion != A2AConformanceSchemaV01 {
		return A2AConformanceManifestV02{}, fmt.Errorf("A2A conformance manifest schemaVersion = %q, want %q", schemaVersion, A2AConformanceSchemaV01)
	}
	if profileSchemaVersion != A2AProfileSchemaVersionV02 {
		return A2AConformanceManifestV02{}, fmt.Errorf("A2A conformance manifest profileSchemaVersion = %q, want %q", profileSchemaVersion, A2AProfileSchemaVersionV02)
	}
	if protocolVersion != A2AProfileProtocolVersion {
		return A2AConformanceManifestV02{}, fmt.Errorf("A2A conformance manifest protocolVersion = %q, want %q", protocolVersion, A2AProfileProtocolVersion)
	}
	if len(wireCases) == 0 {
		return A2AConformanceManifestV02{}, fmt.Errorf("A2A conformance manifest cases must not be empty")
	}

	manifest := A2AConformanceManifestV02{
		SchemaVersion:        schemaVersion,
		ProfileSchemaVersion: profileSchemaVersion,
		ProtocolVersion:      protocolVersion,
		Cases:                make([]A2AConformanceCaseV02, 0, len(wireCases)),
	}
	caseIDs := make(map[string]struct{}, len(wireCases))
	for index, wireCase := range wireCases {
		manifestCase, err := decodeA2AConformanceCaseV02(index, wireCase)
		if err != nil {
			return A2AConformanceManifestV02{}, err
		}
		if _, exists := caseIDs[manifestCase.ID]; exists {
			return A2AConformanceManifestV02{}, fmt.Errorf("A2A conformance manifest contains duplicate case id %q", manifestCase.ID)
		}
		caseIDs[manifestCase.ID] = struct{}{}
		manifest.Cases = append(manifest.Cases, manifestCase)
	}
	for _, manifestCase := range manifest.Cases {
		caseLabel := fmt.Sprintf("A2A conformance case %q", manifestCase.ID)
		if err := validateA2AConformanceRegularFile(corpus, manifestCase.File); err != nil {
			return A2AConformanceManifestV02{}, fmt.Errorf("%s file: %w", caseLabel, err)
		}
		if manifestCase.RequestFile != "" {
			if err := validateA2AConformanceRegularFile(corpus, manifestCase.RequestFile); err != nil {
				return A2AConformanceManifestV02{}, fmt.Errorf("%s requestFile: %w", caseLabel, err)
			}
		}
	}
	return manifest, nil
}

func decodeA2AConformanceCaseV02(index int, wireCase a2aConformanceCaseJSON) (A2AConformanceCaseV02, error) {
	caseLabel := fmt.Sprintf("A2A conformance case %d", index)
	id, err := requireA2AManifestField(caseLabel+" id", wireCase.ID)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	if id == "" {
		return A2AConformanceCaseV02{}, fmt.Errorf("%s id must not be empty", caseLabel)
	}
	caseLabel = fmt.Sprintf("A2A conformance case %q", id)

	file, err := requireA2AManifestField(caseLabel+" file", wireCase.File)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	operation, err := requireA2AManifestField(caseLabel+" operation", wireCase.Operation)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	fixtureKind, err := requireA2AManifestField(caseLabel+" fixtureKind", wireCase.FixtureKind)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	expectedValid, err := requireA2AManifestField(caseLabel+" expectedValid", wireCase.ExpectedValid)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	mediaType, err := requireA2AManifestField(caseLabel+" mediaType", wireCase.MediaType)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	rules, err := requireA2AManifestField(caseLabel+" rules", wireCase.Rules)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	requestFile, hasRequestFile, err := optionalA2AManifestField(caseLabel+" requestFile", wireCase.RequestFile)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	wireResultKind, hasWireResultKind, err := optionalA2AManifestField(caseLabel+" wireResultKind", wireCase.WireResultKind)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	goConcreteType, hasGoConcreteType, err := optionalA2AManifestField(caseLabel+" goConcreteType", wireCase.GoConcreteType)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}
	protocolError, hasProtocolError, err := optionalA2AManifestField(caseLabel+" protocolError", wireCase.ProtocolError)
	if err != nil {
		return A2AConformanceCaseV02{}, err
	}

	for fieldName, value := range map[string]string{
		"file":        file,
		"operation":   operation,
		"fixtureKind": fixtureKind,
		"mediaType":   mediaType,
	} {
		if value == "" {
			return A2AConformanceCaseV02{}, fmt.Errorf("%s %s must not be empty", caseLabel, fieldName)
		}
	}
	for fieldName, field := range map[string]struct {
		value   string
		present bool
	}{
		"requestFile":    {requestFile, hasRequestFile},
		"wireResultKind": {wireResultKind, hasWireResultKind},
		"goConcreteType": {goConcreteType, hasGoConcreteType},
	} {
		if field.present && field.value == "" {
			return A2AConformanceCaseV02{}, fmt.Errorf("%s %s must not be empty", caseLabel, fieldName)
		}
	}
	if hasProtocolError && protocolError == "" {
		return A2AConformanceCaseV02{}, fmt.Errorf("%s protocolError must not be empty", caseLabel)
	}
	if len(rules) == 0 {
		return A2AConformanceCaseV02{}, fmt.Errorf("%s rules must not be empty", caseLabel)
	}
	if err := validateA2AConformanceFixturePath(file); err != nil {
		return A2AConformanceCaseV02{}, fmt.Errorf("%s file: %w", caseLabel, err)
	}
	if hasRequestFile {
		if err := validateA2AConformanceFixturePath(requestFile); err != nil {
			return A2AConformanceCaseV02{}, fmt.Errorf("%s requestFile: %w", caseLabel, err)
		}
		if requestFile == file {
			return A2AConformanceCaseV02{}, fmt.Errorf("%s uses its fixture as requestFile", caseLabel)
		}
	}

	manifestCase := A2AConformanceCaseV02{
		ID:             id,
		File:           file,
		RequestFile:    requestFile,
		Operation:      operation,
		FixtureKind:    fixtureKind,
		ExpectedValid:  expectedValid,
		WireResultKind: wireResultKind,
		GoConcreteType: goConcreteType,
		ProtocolError:  protocolError,
		MediaType:      mediaType,
		Rules:          rules,
	}
	if err := validateA2AConformanceCaseMetadata(manifestCase, hasRequestFile, hasWireResultKind, hasGoConcreteType, hasProtocolError); err != nil {
		return A2AConformanceCaseV02{}, err
	}
	return manifestCase, nil
}

func validateA2AConformanceCaseMetadata(manifestCase A2AConformanceCaseV02, hasRequestFile, hasWireResultKind, hasGoConcreteType, hasProtocolError bool) error {
	caseLabel := fmt.Sprintf("A2A conformance case %q", manifestCase.ID)
	wantsRequestFile := manifestCase.FixtureKind == "response" || manifestCase.FixtureKind == "error" || manifestCase.FixtureKind == "stream"
	if wantsRequestFile != hasRequestFile {
		if wantsRequestFile {
			return fmt.Errorf("%s is missing requestFile", caseLabel)
		}
		return fmt.Errorf("%s fixtureKind %q forbids requestFile", caseLabel, manifestCase.FixtureKind)
	}

	switch manifestCase.FixtureKind {
	case "request":
		if !isA2AOperation(manifestCase.Operation) {
			return fmt.Errorf("%s request operation %q is unsupported", caseLabel, manifestCase.Operation)
		}
	case "response":
		if manifestCase.Operation != "message/send" && manifestCase.Operation != "tasks/get" && manifestCase.Operation != "tasks/cancel" {
			return fmt.Errorf("%s response operation %q is unsupported", caseLabel, manifestCase.Operation)
		}
	case "error":
		if manifestCase.Operation != "tasks/get" && manifestCase.Operation != "tasks/cancel" {
			return fmt.Errorf("%s error operation %q is unsupported", caseLabel, manifestCase.Operation)
		}
	case "stream":
		if manifestCase.Operation != "message/stream" {
			return fmt.Errorf("%s stream operation %q is unsupported", caseLabel, manifestCase.Operation)
		}
	case "headers":
		if manifestCase.Operation != "context/propagation" {
			return fmt.Errorf("%s headers operation %q is unsupported", caseLabel, manifestCase.Operation)
		}
	default:
		return fmt.Errorf("%s fixtureKind %q is unsupported", caseLabel, manifestCase.FixtureKind)
	}

	wantMediaType := "application/json"
	wantExtension := ".json"
	if manifestCase.FixtureKind == "stream" {
		wantMediaType = "text/event-stream"
		wantExtension = ".sse"
	}
	if manifestCase.MediaType != wantMediaType {
		return fmt.Errorf("%s mediaType = %q, want %q", caseLabel, manifestCase.MediaType, wantMediaType)
	}
	if path.Ext(manifestCase.File) != wantExtension {
		return fmt.Errorf("%s file extension = %q, want %q", caseLabel, path.Ext(manifestCase.File), wantExtension)
	}
	if hasRequestFile && path.Ext(manifestCase.RequestFile) != ".json" {
		return fmt.Errorf("%s requestFile must be a JSON fixture", caseLabel)
	}

	wantsResultType := manifestCase.FixtureKind == "response" && manifestCase.ExpectedValid
	if wantsResultType != hasWireResultKind || wantsResultType != hasGoConcreteType {
		if wantsResultType {
			return fmt.Errorf("%s valid response requires wireResultKind and goConcreteType", caseLabel)
		}
		return fmt.Errorf("%s must not declare wireResultKind or goConcreteType", caseLabel)
	}
	if wantsResultType {
		switch manifestCase.WireResultKind {
		case "message":
			if manifestCase.GoConcreteType != "*a2a.Message" {
				return fmt.Errorf("%s message result goConcreteType = %q, want *a2a.Message", caseLabel, manifestCase.GoConcreteType)
			}
			if manifestCase.Operation != "message/send" {
				return fmt.Errorf("%s operation %q cannot return a Message result", caseLabel, manifestCase.Operation)
			}
		case "task":
			if manifestCase.GoConcreteType != "*a2a.Task" {
				return fmt.Errorf("%s task result goConcreteType = %q, want *a2a.Task", caseLabel, manifestCase.GoConcreteType)
			}
		default:
			return fmt.Errorf("%s wireResultKind %q is unsupported", caseLabel, manifestCase.WireResultKind)
		}
	}

	wantsProtocolError := manifestCase.FixtureKind == "error" || !manifestCase.ExpectedValid
	if wantsProtocolError != hasProtocolError {
		if wantsProtocolError {
			return fmt.Errorf("%s requires protocolError", caseLabel)
		}
		return fmt.Errorf("%s must not declare protocolError", caseLabel)
	}

	ruleSet := make(map[A2AConformanceRuleIDV02]struct{}, len(manifestCase.Rules))
	for _, ruleID := range manifestCase.Rules {
		if !isKnownA2AConformanceRuleV02(ruleID) {
			return fmt.Errorf("%s contains unknown rule %q", caseLabel, ruleID)
		}
		if _, exists := ruleSet[ruleID]; exists {
			return fmt.Errorf("%s repeats rule %q", caseLabel, ruleID)
		}
		if !isA2AConformanceRuleAllowed(manifestCase, ruleID) {
			return fmt.Errorf("%s rule %q does not apply to %s %s", caseLabel, ruleID, manifestCase.Operation, manifestCase.FixtureKind)
		}
		ruleSet[ruleID] = struct{}{}
	}
	for _, requiredRule := range requiredA2AConformanceRules(manifestCase) {
		if _, exists := ruleSet[requiredRule]; !exists {
			return fmt.Errorf("%s is missing required rule %q", caseLabel, requiredRule)
		}
	}
	if hasProtocolError {
		if !isKnownA2AProtocolErrorClaim(manifestCase.ProtocolError) {
			return fmt.Errorf("%s protocolError %q is unsupported", caseLabel, manifestCase.ProtocolError)
		}
		if !a2aProtocolErrorHasClaimedRule(manifestCase.ProtocolError, ruleSet) {
			return fmt.Errorf("%s protocolError %q has no corresponding rule claim", caseLabel, manifestCase.ProtocolError)
		}
	}
	return nil
}

func requiredA2AConformanceRules(manifestCase A2AConformanceCaseV02) []A2AConformanceRuleIDV02 {
	rules := make([]A2AConformanceRuleIDV02, 0, 12)
	if manifestCase.FixtureKind == "response" || manifestCase.FixtureKind == "error" || manifestCase.FixtureKind == "stream" {
		rules = append(rules, A2ARuleJSONRPCEnvelope, A2ARuleResultXORError)
	}
	if !manifestCase.ExpectedValid {
		return rules
	}
	switch manifestCase.FixtureKind {
	case "request":
		rules = append(rules, A2ARuleJSONRPCEnvelope, A2ARuleRequestParams)
		if manifestCase.Operation == "tasks/get" {
			rules = append(rules, A2ARuleHistoryLength)
		}
		return rules
	case "response":
		rules = append(rules, A2ARuleRequestResponseID, A2ARuleResultType)
		if manifestCase.Operation == "message/send" {
			rules = append(rules, A2ARuleResultUnion)
		}
		if manifestCase.WireResultKind == "message" {
			return append(rules, A2ARuleMessageResult)
		}
		rules = append(rules, A2ARuleTaskIdentity, A2ARuleTaskState)
		if manifestCase.Operation == "tasks/get" {
			rules = append(rules, A2ARuleHistoryLength)
		}
		if manifestCase.Operation == "tasks/cancel" {
			rules = append(rules, A2ARuleSameTask, A2ARuleCanceledState)
		}
		return rules
	case "error":
		return append(rules, A2ARuleRequestResponseID, A2ARuleErrorOnly)
	case "stream":
		return append(rules,
			A2ARuleSSEFraming,
			A2ARuleRequestResponseID,
			A2ARuleEventKinds,
			A2ARuleTaskIdentity,
			A2ARuleTaskState,
			A2ARuleTaskContextStability,
			A2ARuleTerminalRequired,
			A2ARuleTerminalLast,
			A2ARuleArtifactOrder,
			A2ARuleArtifactLastChunk,
		)
	case "headers":
		return []A2AConformanceRuleIDV02{A2ARuleFiveContextHeaders}
	default:
		return rules
	}
}

func isA2AConformanceRuleAllowed(manifestCase A2AConformanceCaseV02, ruleID A2AConformanceRuleIDV02) bool {
	switch ruleID {
	case A2ARuleJSONRPCEnvelope:
		return manifestCase.FixtureKind != "headers"
	case A2ARuleRequestParams:
		return manifestCase.FixtureKind == "request" && isA2AOperation(manifestCase.Operation)
	case A2ARuleRequestResponseID, A2ARuleResultXORError:
		return manifestCase.FixtureKind == "response" || manifestCase.FixtureKind == "error" || manifestCase.FixtureKind == "stream"
	case A2ARuleResultUnion, A2ARuleMessageResult:
		return manifestCase.FixtureKind == "response" && manifestCase.Operation == "message/send"
	case A2ARuleResultType:
		return manifestCase.FixtureKind == "response" && manifestCase.ExpectedValid
	case A2ARuleTaskIdentity, A2ARuleTaskState:
		return manifestCase.FixtureKind == "stream" || manifestCase.FixtureKind == "response" && manifestCase.Operation != "message/send" || manifestCase.FixtureKind == "response" && manifestCase.WireResultKind == "task"
	case A2ARuleSSEFraming, A2ARuleEventKinds, A2ARuleTaskContextStability, A2ARuleTerminalRequired, A2ARuleTerminalLast, A2ARuleArtifactOrder, A2ARuleArtifactLastChunk:
		return manifestCase.FixtureKind == "stream" && manifestCase.Operation == "message/stream"
	case A2ARuleHistoryLength, A2ARuleRejectedMapping, A2ARuleUnsupportedStateMapping:
		return manifestCase.Operation == "tasks/get" && (manifestCase.FixtureKind == "request" || manifestCase.FixtureKind == "response")
	case A2ARuleErrorOnly:
		return manifestCase.FixtureKind == "error"
	case A2ARuleSameTask, A2ARuleCanceledState:
		return manifestCase.FixtureKind == "response" && manifestCase.Operation == "tasks/cancel"
	case A2ARuleFiveContextHeaders:
		return manifestCase.FixtureKind == "headers" && manifestCase.Operation == "context/propagation"
	default:
		return false
	}
}

func isKnownA2AConformanceRuleV02(ruleID A2AConformanceRuleIDV02) bool {
	switch ruleID {
	case A2ARuleJSONRPCEnvelope,
		A2ARuleRequestParams,
		A2ARuleRequestResponseID,
		A2ARuleResultXORError,
		A2ARuleResultUnion,
		A2ARuleResultType,
		A2ARuleMessageResult,
		A2ARuleTaskIdentity,
		A2ARuleTaskState,
		A2ARuleSSEFraming,
		A2ARuleEventKinds,
		A2ARuleTaskContextStability,
		A2ARuleTerminalRequired,
		A2ARuleTerminalLast,
		A2ARuleArtifactOrder,
		A2ARuleArtifactLastChunk,
		A2ARuleHistoryLength,
		A2ARuleErrorOnly,
		A2ARuleRejectedMapping,
		A2ARuleUnsupportedStateMapping,
		A2ARuleSameTask,
		A2ARuleCanceledState,
		A2ARuleFiveContextHeaders:
		return true
	default:
		return false
	}
}

func isKnownA2AProtocolErrorClaim(protocolError A2AProtocolErrorV02) bool {
	switch protocolError {
	case A2AProtocolErrorInvalidJSONRPCEnvelope,
		A2AProtocolErrorInvalidJSONRPCVersion,
		A2AProtocolErrorInvalidResponseIDType,
		A2AProtocolErrorResponseIDMismatch,
		A2AProtocolErrorResultErrorExclusivity,
		A2AProtocolErrorResultErrorRequired,
		A2AProtocolErrorInvalidResultKind,
		A2AProtocolErrorInvalidMessageResult,
		A2AProtocolErrorInvalidTask,
		A2AProtocolErrorUnsupportedTaskState,
		A2AProtocolErrorTaskContextMismatch,
		A2AProtocolErrorEventAfterTerminal,
		A2AProtocolErrorEOFWithoutTerminal,
		A2AProtocolErrorArtifactAppendWithoutBase,
		A2AProtocolErrorArtifactAfterLastChunk,
		A2AProtocolErrorTaskNotFound,
		A2AProtocolErrorTaskNotCancelable:
		return true
	default:
		return false
	}
}

func a2aProtocolErrorHasClaimedRule(protocolError A2AProtocolErrorV02, rules map[A2AConformanceRuleIDV02]struct{}) bool {
	var candidates []A2AConformanceRuleIDV02
	switch protocolError {
	case A2AProtocolErrorInvalidJSONRPCEnvelope, A2AProtocolErrorInvalidJSONRPCVersion, A2AProtocolErrorInvalidResponseIDType:
		candidates = []A2AConformanceRuleIDV02{A2ARuleJSONRPCEnvelope}
	case A2AProtocolErrorResponseIDMismatch:
		candidates = []A2AConformanceRuleIDV02{A2ARuleRequestResponseID}
	case A2AProtocolErrorResultErrorExclusivity, A2AProtocolErrorResultErrorRequired:
		candidates = []A2AConformanceRuleIDV02{A2ARuleResultXORError}
	case A2AProtocolErrorInvalidResultKind:
		candidates = []A2AConformanceRuleIDV02{A2ARuleResultUnion}
	case A2AProtocolErrorInvalidMessageResult:
		candidates = []A2AConformanceRuleIDV02{A2ARuleMessageResult}
	case A2AProtocolErrorTaskContextMismatch:
		candidates = []A2AConformanceRuleIDV02{A2ARuleTaskContextStability}
	case A2AProtocolErrorEventAfterTerminal:
		candidates = []A2AConformanceRuleIDV02{A2ARuleTerminalLast}
	case A2AProtocolErrorEOFWithoutTerminal:
		candidates = []A2AConformanceRuleIDV02{A2ARuleTerminalRequired}
	case A2AProtocolErrorArtifactAppendWithoutBase:
		candidates = []A2AConformanceRuleIDV02{A2ARuleArtifactOrder}
	case A2AProtocolErrorArtifactAfterLastChunk:
		candidates = []A2AConformanceRuleIDV02{A2ARuleArtifactOrder, A2ARuleArtifactLastChunk}
	case A2AProtocolErrorTaskNotFound, A2AProtocolErrorTaskNotCancelable:
		candidates = []A2AConformanceRuleIDV02{A2ARuleErrorOnly}
	case A2AProtocolErrorInvalidTask:
		candidates = []A2AConformanceRuleIDV02{A2ARuleTaskIdentity, A2ARuleTaskState}
	case A2AProtocolErrorUnsupportedTaskState:
		candidates = []A2AConformanceRuleIDV02{A2ARuleTaskState, A2ARuleUnsupportedStateMapping}
	}
	for _, candidate := range candidates {
		if _, exists := rules[candidate]; exists {
			return true
		}
	}
	return false
}

func isA2AOperation(operation string) bool {
	switch operation {
	case "message/send", "message/stream", "tasks/get", "tasks/cancel":
		return true
	default:
		return false
	}
}

func requireA2AManifestField[T any](name string, field a2aManifestField[T]) (T, error) {
	if !field.Present {
		var zero T
		return zero, fmt.Errorf("%s is missing", name)
	}
	if field.Null {
		var zero T
		return zero, fmt.Errorf("%s must not be null", name)
	}
	return field.Value, nil
}

func optionalA2AManifestField[T any](name string, field a2aManifestField[T]) (T, bool, error) {
	if !field.Present {
		var zero T
		return zero, false, nil
	}
	if field.Null {
		var zero T
		return zero, false, fmt.Errorf("%s must not be null", name)
	}
	return field.Value, true, nil
}

func readA2AConformanceFixtureV02(corpus fs.FS, fixturePath string) ([]byte, error) {
	if err := validateA2AConformanceFixturePath(fixturePath); err != nil {
		return nil, fmt.Errorf("A2A conformance fixture %q: %w", fixturePath, err)
	}
	if err := validateA2AConformanceRegularFile(corpus, fixturePath); err != nil {
		return nil, fmt.Errorf("A2A conformance fixture %q: %w", fixturePath, err)
	}
	data, err := fs.ReadFile(corpus, fixturePath)
	if err != nil {
		return nil, fmt.Errorf("read A2A conformance fixture %q: %w", fixturePath, err)
	}
	return data, nil
}

func validateA2AConformanceRegularFile(corpus fs.FS, fixturePath string) error {
	info, err := fs.Lstat(corpus, fixturePath)
	if err != nil {
		return fmt.Errorf("stat fixture: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("fixture is not a regular file")
	}
	return nil
}

func validateA2AConformanceFixturePath(fixturePath string) error {
	if fixturePath == "" {
		return fmt.Errorf("fixture path must not be empty")
	}
	if !fs.ValidPath(fixturePath) || path.Clean(fixturePath) != fixturePath {
		return fmt.Errorf("fixture path must be a canonical relative path")
	}
	if strings.Contains(fixturePath, "\\") {
		return fmt.Errorf("fixture path must use forward slashes")
	}
	if containsA2AASCIIControl(fixturePath) {
		return fmt.Errorf("fixture path contains an ASCII control character")
	}
	if strings.ContainsAny(fixturePath, "%?#<>\"|*") {
		return fmt.Errorf("fixture path contains a noncanonical character")
	}
	if hasA2AURIScheme(fixturePath) {
		return fmt.Errorf("fixture path must not contain a URI scheme")
	}
	if strings.ContainsRune(fixturePath, ':') {
		return fmt.Errorf("fixture path contains a nonportable colon")
	}
	for _, segment := range strings.Split(fixturePath, "/") {
		if strings.TrimRight(segment, " .") != segment {
			return fmt.Errorf("fixture path contains a platform-equivalent segment")
		}
		if isA2AWindowsReservedBasename(segment) {
			return fmt.Errorf("fixture path contains a Windows reserved device basename")
		}
	}
	return nil
}

func containsA2AASCIIControl(value string) bool {
	for _, character := range value {
		if character <= 0x1f || character == 0x7f {
			return true
		}
	}
	return false
}

func isA2AWindowsReservedBasename(segment string) bool {
	basename := segment
	if extension := strings.IndexByte(segment, '.'); extension >= 0 {
		basename = segment[:extension]
	}
	basename = strings.ToUpper(basename)
	if basename == "CON" || basename == "PRN" || basename == "AUX" || basename == "NUL" {
		return true
	}
	if len(basename) != 4 || basename[3] < '1' || basename[3] > '9' {
		return false
	}
	return basename[:3] == "COM" || basename[:3] == "LPT"
}

func hasA2AURIScheme(value string) bool {
	colon := strings.IndexByte(value, ':')
	if colon <= 0 {
		return false
	}
	if slash := strings.IndexByte(value, '/'); slash >= 0 && slash < colon {
		return false
	}
	for index := 0; index < colon; index++ {
		character := value[index]
		if index == 0 {
			if !isA2AASCIIAlpha(character) {
				return false
			}
			continue
		}
		if !isA2AASCIIAlpha(character) && (character < '0' || character > '9') && character != '+' && character != '-' && character != '.' {
			return false
		}
	}
	return true
}

func isA2AASCIIAlpha(character byte) bool {
	return character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
}

func MapA2ATaskState(state a2a.TaskState) (A2ATaskStateMapping, error) {
	switch state {
	case a2a.TaskStateSubmitted, a2a.TaskStateWorking:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTransient,
			InvocationStatus: "running",
		}, nil
	case a2a.TaskStateCompleted:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTerminal,
			InvocationStatus: "succeeded",
		}, nil
	case a2a.TaskStateFailed, a2a.TaskStateRejected:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTerminal,
			InvocationStatus: "failed",
			ErrorCode:        ErrorCodeAgentExecutionFailed,
		}, nil
	case a2a.TaskStateCanceled:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTerminal,
			InvocationStatus: "canceled",
			ErrorCode:        ErrorCodeCanceled,
		}, nil
	case a2a.TaskStateAuthRequired, a2a.TaskStateInputRequired, a2a.TaskStateUnknown:
		return A2ATaskStateMapping{}, &A2AProfileStateError{
			State:     state,
			Reason:    "recognized but unsupported",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	case a2a.TaskStateUnspecified:
		return A2ATaskStateMapping{}, &A2AProfileStateError{
			State:     state,
			Reason:    "unspecified",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	default:
		return A2ATaskStateMapping{}, &A2AProfileStateError{
			State:     state,
			Reason:    "not defined by A2A protocol 0.3.0",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
}

func ValidateA2ATask(task *a2a.Task) (A2ATaskStateMapping, error) {
	if task == nil {
		return A2ATaskStateMapping{}, &A2AProfileTaskError{
			Reason:    "task is missing",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	if task.ID == "" {
		return A2ATaskStateMapping{}, &A2AProfileTaskError{
			Reason:    "task id is empty",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	if task.ContextID == "" {
		return A2ATaskStateMapping{}, &A2AProfileTaskError{
			Reason:    "context id is empty",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	return MapA2ATaskState(task.Status.State)
}

func ValidateA2AMessageResult(message *a2a.Message) error {
	if message == nil {
		return &A2AProfileMessageError{
			Reason:    "message is missing",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	if message.ID == "" {
		return &A2AProfileMessageError{
			Reason:    "message id is empty",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	if message.Role != a2a.MessageRoleAgent {
		return &A2AProfileMessageError{
			Reason:    "message role is not agent",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	if len(message.Parts) == 0 {
		return &A2AProfileMessageError{
			Reason:    "message has no parts",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	return nil
}

func decodeA2AProfileV02[T any](path string) (T, error) {
	var value T
	data, err := a2aProfileV02Files.ReadFile(path)
	if err != nil {
		return value, fmt.Errorf("read %s: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return value, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}

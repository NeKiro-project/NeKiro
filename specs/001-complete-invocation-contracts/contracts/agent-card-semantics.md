# Contract Design: Agent Card Semantic Conformance

## Contract Composition

Agent Card `0.2` conformance requires both:

1. Structural validation against
   `contracts/schemas/agent-card.v0.2.schema.json`.
2. Semantic validation against
   `contracts/agent-card/v0.2/semantic-rules.md`.

Passing only one layer is invalid.

## Normative Rules

- `AC-SEM-001`: every `skills[*].id` MUST be unique within one Card.
- `AC-SEM-002`: every `permissions[*].id` MUST be unique within one Card.
- `AC-SEM-003`: every `skills[*].requiredPermissions[*]` MUST exactly match a
  `permissions[*].id` declared in the same Card version.

Comparison is case-sensitive JSON string equality. A declaration in another
Agent Card version cannot satisfy a reference.

## Portable Conformance Corpus

Directory: `contracts/agent-card/v0.2/conformance/`

Minimum raw fixtures:

- valid baseline;
- valid multiple skills sharing one declared permission;
- invalid duplicate skill ID on otherwise distinct skill objects;
- invalid duplicate permission ID on otherwise distinct declarations;
- invalid undeclared required permission;
- invalid permission declared only in another Card version.

`manifest.json` records stable case ID, fixture path, expected validity, and
violated rule IDs. Fixtures are authored independently of Go marshaling.

## Go Mapping

Go retains structural-then-semantic validation. Semantic errors expose stable
rule IDs for conformance tests; wording and evaluation order are not public
contract. The Go implementation does not parse Markdown or execute a new rules
DSL.

## Migration

The stricter semantic contract is Agent Card `0.2`. Version `0.1` remains
historical and is not silently amended. The first Registry implementation
accepts only active `0.2`, because no published Registry data exists to justify
a dual-version compatibility path.

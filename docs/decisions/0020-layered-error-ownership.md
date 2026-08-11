# ADR 0020: Layered Error Ownership

- Status: Proposed for issue #116
- Date: 2026-08-11
- Decision owner: Platform Architecture
- Related: `nekiro-sdk-go#3`, `nekiro-sdk-go#4`, `NeKiro-Samples#11`, and
  `NeKiro-Samples#12`

## Context

Core domain packages, cross-process contracts, and Agent Runtime lifecycle
code need different error semantics. Treating them as one global taxonomy
would couple independent owners and would make dependency details easier to
leak across trust boundaries.

The Runtime lifecycle also repeats setup, serving, lease observation, and
shutdown handling in each Sample. The SDK needs to provide that lifecycle
mechanism without taking ownership of Runtime-specific configuration,
handlers, transport security, or policy.

## Proposed Direction

This ADR will define:

1. owner-local typed errors for Core domain packages;
2. stable wire semantics through versioned contracts;
3. explicit error translation at process and protocol boundaries;
4. safe Runtime lifecycle stages owned by `nekiro-sdk-go/agent/host`; and
5. the rejection of a global catch mechanism or flattened Core error
   taxonomy.

The final decision and consequences will be completed on the draft pull
request before it is marked ready for review.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.

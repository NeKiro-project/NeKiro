# Clarify: Trusted Agent Publication

No blocking clarification questions remain. The following decisions are
recorded from the project charter and the requested product direction:

1. Provider identity is a first-class Registry fact rather than another free
   text field on Agent Card owner metadata.
2. HTTP well-known challenge is the first endpoint proof method. DNS and
   organization attestations are intentionally deferred.
3. Endpoint verification denies private/loopback/link-local destinations by
   default. Development exceptions must be explicit configuration, because
   adding a localhost fallback would violate the repository's no-unsafe-
   fallback policy.
4. Verification failure is a typed failure, not an empty verification result.
5. A changed Card or endpoint creates a new immutable release; historical
   Ledger identity is never rewritten.

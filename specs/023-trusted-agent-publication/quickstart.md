# Trusted publication quickstart (Slice A)

1. Authenticate as the provider owner at the Gateway.
2. Create an endpoint binding for the exact A2A endpoint.
3. Create a single-use challenge and place the returned proof at the
   challenge URL's documented well-known location on the provider endpoint.
4. Complete the challenge. Inspect the binding until it is `verified`.
5. Continue with the release lifecycle in Sub-Issue #49. An endpoint binding
   alone does not publish or install an Agent.

The service does not retry a failed challenge, follow redirects, or select a
different endpoint. Correct the provider endpoint or explicitly change the
approved network policy, then create a new challenge.

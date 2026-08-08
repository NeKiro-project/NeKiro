// Package gateway defines the provider-neutral external Gateway foundation.
//
// It describes immutable desired routes for exact Catalog releases and the
// narrow provider contract used to reconcile and observe them. It does not
// proxy requests, select instances, own credentials, or establish data-plane
// readiness.
package gateway

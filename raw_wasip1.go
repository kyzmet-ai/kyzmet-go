//go:build wasip1

package raw

// Kyzmet Go guests are WASI Preview 2 components. A wasip1 build fails here on
// purpose: build with `tinygo build -target=wasip2`.
var _ = kyzmetGoGuestsRequireWasip2

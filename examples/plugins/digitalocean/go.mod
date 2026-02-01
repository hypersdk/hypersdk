module hypersdk/plugins/digitalocean

go 1.24

// This module must be built as a plugin
// Run: go build -buildmode=plugin -o digitalocean.so

// Important: This plugin MUST be compiled with the EXACT same Go version
// as the hypervisord daemon to ensure ABI compatibility

require (
	hypersdk v0.0.1
)

replace hypersdk => ../../..

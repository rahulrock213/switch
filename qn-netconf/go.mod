module qn-netconf

go 1.22.0 // Or your actual Go version, e.g., 1.18, 1.19, etc.

toolchain go1.22.2

require golang.org/x/crypto v0.22.0 // This is an example version

require (
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/gosnmp/gosnmp v1.40.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
)

// If you have other external dependencies, they would be listed here.
// Running `go mod tidy` later will populate this correctly.

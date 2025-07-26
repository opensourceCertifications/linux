module chaos-agent

go 1.24.2

toolchain go1.24.5

require (
	github.com/opensourceCertifications/linux/shared v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.12.0
)

require golang.org/x/sys v0.11.0 // indirect

replace github.com/opensourceCertifications/linux/shared => ./shared

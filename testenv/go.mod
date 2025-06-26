module testenv

go 1.24.2

toolchain go1.24.4

require (
	github.com/opensourceCertifications/linux/shared v0.0.0
	github.com/pquerna/otp v1.4.0
)

require github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect

replace github.com/opensourceCertifications/linux/shared => ../shared

module monitor

go 1.24.2

toolchain go1.24.4

require github.com/pquerna/otp v1.4.0

require (
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	shared v0.0.0-00010101000000-000000000000 // indirect
)

replace shared => ../shared

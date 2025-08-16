module github.com/BRUHItsABunny/Premiumize-File-Sync

go 1.23.0

toolchain go1.24.1

replace (
	github.com/cornelk/hashmap v1.0.8 => github.com/BRUHItsABunny/hashmap v0.0.0-20221125164545-8b59f13d589a
	go.uber.org/atomic v1.10.0 => github.com/BRUHItsABunny/atomic v0.0.0-20221125214309-9e798cd18888
)

require (
	github.com/joho/godotenv v1.5.1
	go.uber.org/atomic v1.11.0
)

require (
	github.com/BRUHItsABunny/bunnlog v0.0.1
	github.com/BRUHItsABunny/bunterm v0.0.2
	github.com/BRUHItsABunny/gOkHttp v0.3.8
	github.com/BRUHItsABunny/gOkHttp-download v0.3.6
	github.com/BRUHItsABunny/go-ghvu v0.0.3
	github.com/BRUHItsABunny/go-premiumize v0.0.2
	github.com/cornelk/hashmap v1.0.8
	github.com/davecgh/go-spew v1.1.1
	github.com/dustin/go-humanize v1.0.1
	golang.org/x/sync v0.16.0
)

require (
	github.com/etherlabsio/go-m3u8 v1.0.0 // indirect
	github.com/google/go-github/v48 v48.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/yapingcat/gomedia v0.0.0-20240906162731-17feea57090c // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/term v0.34.0 // indirect
	golang.org/x/text v0.28.0 // indirect
)

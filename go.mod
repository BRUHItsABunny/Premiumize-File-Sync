module github.com/BRUHItsABunny/Premiumize-File-Sync

go 1.19

replace (
	github.com/cornelk/hashmap v1.0.8 => github.com/BRUHItsABunny/hashmap v0.0.0-20221125164545-8b59f13d589a
	go.uber.org/atomic v1.10.0 => github.com/BRUHItsABunny/atomic v0.0.0-20221125214309-9e798cd18888
)

require (
	github.com/joho/godotenv v1.4.0
	go.uber.org/atomic v1.10.0
)

require (
	github.com/BRUHItsABunny/bunnlog v0.0.1
	github.com/BRUHItsABunny/bunterm v0.0.2
	github.com/BRUHItsABunny/go-ghvu v0.0.3
	github.com/BRUHItsABunny/go-premiumize v0.0.2
	github.com/cornelk/hashmap v1.0.8
	github.com/davecgh/go-spew v1.1.1
	github.com/dustin/go-humanize v1.0.0
)

require (
	github.com/google/go-github/v48 v48.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	golang.org/x/crypto v0.4.0 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/term v0.3.0 // indirect
)

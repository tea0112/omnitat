module github.com/tea0112/omnitat/services/go/iam/identity

go 1.26.1

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.7.3
	github.com/tea0112/omnitat/libs/go v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/lib/pq v1.12.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
)

replace github.com/tea0112/omnitat/libs/go => ../../../../libs/go

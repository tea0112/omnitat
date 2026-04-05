module github.com/tea0112/omnitat/services/go/iam/identity

go 1.26.1

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/tea0112/omnitat/libs/go v0.0.0
)

require (
	github.com/lib/pq v1.12.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
)

replace github.com/tea0112/omnitat/libs/go => ../../../../libs/go

module github.com/mlctrez/factop

go 1.24.2

replace github.com/mlctrez/rcon => ../rcon

require (
	github.com/google/uuid v1.6.0
	github.com/kardianos/service v1.2.4
	github.com/magefile/mage v1.15.0
	github.com/mlctrez/bind v1.0.2
	github.com/mlctrez/rcon v1.0.1
	github.com/mlctrez/servicego v1.4.10
	github.com/nats-io/nats-server/v2 v2.11.6
	github.com/nats-io/nats.go v1.43.0
)

require (
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/gorcon/rcon v1.4.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/nats-io/jwt/v2 v2.7.4 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)

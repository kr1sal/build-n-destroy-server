## Generate pb

```bash
protoc --go_out=. --go-grpc_out=. --go_opt=module=kr1sal.com/build-n-destroy --go-grpc_opt=module=kr1sal.com/build-n-destroy  ./proto/game.proto
```

## Start server

```bash
go run ./cmd/game/main.go
```

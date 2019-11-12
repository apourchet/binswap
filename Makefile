default: bin/binswap

bin/binswap:
	mkdir -p bin
	GOOS=linux CGO_ENABLED=0 go build -i -ldflags "-s" -o bin/binswap ./*.go

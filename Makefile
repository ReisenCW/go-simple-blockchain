BINARY := go-blockchain

build:
	@echo "====> Go build"
	@go build -o $(BINARY)

.PHONY: build
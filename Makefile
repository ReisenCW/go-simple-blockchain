BINARY := go-blockchain

build:
	@echo "====> Go build"
	@go build -o $(BINARY)
# 	删除*.dat和*.db文件
	@find . -type f \( -name "*.dat" -o -name "*.db" \) -exec rm -f {} \;

.PHONY: build
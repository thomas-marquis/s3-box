all:
	@echo "Building...."
	@go build -v -o s3box main.go
	@echo "Build complete."

.PHONY: run
run: all
	@echo "Running..."
	@./s3box

.PHONY: clean
clean:
	@rm -f s3box

.PHONY: test
test:
	@go test -v ./...

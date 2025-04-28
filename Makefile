all:
	@go build -o s3box main.go

.PHONY: run
run: all
	@./s3box

.PHONY: clean
clean:
	@rm -f s3box

.PHONY: test
test:
	@go test -v ./...
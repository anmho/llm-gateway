

.PHONY: bulid

build:
	@go build -o ./bin/api ./main.go
.PHONY: clean
clean:
	@rm ./bin/*


.PHONY: run
run:
	@go run .
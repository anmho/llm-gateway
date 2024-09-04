
.PHONY: chat
chat:
	@npm run --prefix ./chat build

.PHONY: api
api:
	@go build -o ./bin/api ./main.go

.PHONY: clean
clean:
	@rm ./bin/*

.PHONY: run
run: chat api
	@go run . -stage development
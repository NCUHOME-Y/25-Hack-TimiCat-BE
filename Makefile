RUN_CMD=go run ./cmd/app
TEST_CMD=go test ./...

.PHONY: run test
run:
	$(RUN_CMD)

test:
	$(TEST_CMD)

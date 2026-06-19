.PHONY: all build run clean test lint docker-build swagger

APP_NAME    := earthquake-api
CMD_DIR     := ./cmd/api
BUILD_DIR   := ./bin
GO_FLAGS    := -ldflags="-s -w"

all: build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(GO_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

run:
	@echo "Running $(APP_NAME)..."
	go run $(CMD_DIR)

run-hot:
	@echo "Running $(APP_NAME) with air (hot reload)..."
	@command -v air >/dev/null 2>&1 || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@echo "Clean complete"

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -n 1

lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 --env-file .env $(APP_NAME):latest

docker-compose-up:
	@echo "Starting with Docker Compose..."
	docker compose up --build -d

docker-compose-down:
	@echo "Stopping Docker Compose..."
	docker compose down

swagger:
	@echo "Generating Swagger docs..."
	@command -v swag >/dev/null 2>&1 || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/api/main.go --parseDependency --parseInternal
	@echo "Swagger docs generated in ./docs"

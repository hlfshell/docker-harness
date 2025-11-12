# 🐳 Docker Harness - Just Commands
# A colorful, emoji-filled command runner for docker-harness

# Default recipe - show all available commands
default:
    @just --list

# 🧪 Run all tests (core + all databases)
test:
    @echo "🧪 Running all tests..."
    @go test -v ./...
    @cd databases/postgres && go test -v ./...
    @cd databases/mysql && go test -v ./...
    @cd databases/redis && go test -v ./...
    @cd databases/memcached && go test -v ./...

# 📊 Show test coverage
coverage:
    @echo "📊 Generating test coverage..."
    @go test -coverprofile=coverage.out ./...
    @go tool cover -html=coverage.out -o coverage.html
    @echo "✅ Coverage report generated: coverage.html"

# 🧹 Clean test artifacts and temporary files
clean:
    @echo "🧹 Cleaning up..."
    @go clean -testcache
    @echo "✅ Cleaned test cache"

# 📦 Build all modules
build:
    @echo "📦 Building all modules..."
    @go build ./...
    @cd databases/postgres && go build ./...
    @cd databases/mysql && go build ./...
    @cd databases/redis && go build ./...
    @cd databases/memcached && go build ./...
    @echo "✅ Build complete"

# 🎨 Format all Go code
format:
    @echo "🎨 Formatting code..."
    @go fmt ./...
    @cd databases/postgres && go fmt ./...
    @cd databases/mysql && go fmt ./...
    @cd databases/redis && go fmt ./...
    @cd databases/memcached && go fmt ./...
    @echo "✅ Formatting complete"

# 🔍 Run go vet on all modules
vet:
    @echo "🔍 Running go vet..."
    @go vet ./...
    @cd databases/postgres && go vet ./...
    @cd databases/mysql && go vet ./...
    @cd databases/redis && go vet ./...
    @cd databases/memcached && go vet ./...
    @echo "✅ Vet complete"

# 📝 Show current version
version:
    @echo "📝 Current version:"
    @git describe --tags --abbrev=0 2>/dev/null || echo "No version tagged yet"

# 🚀 Tag and push a new version (usage: just release 1.2.3)
release version:
    @echo "🚀 Releasing version {{version}}..."
    @git tag -a "v{{version}}" -m "Release v{{version}}"
    @git push origin "v{{version}}"
    @echo "✅ Released v{{version}}!"

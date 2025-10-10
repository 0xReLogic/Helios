# Contributing to Helios

Thank you for your interest in contributing to Helios! This document provides guidelines and instructions for contributing to this project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please be respectful and considerate of others.

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue with the following information:

- A clear, descriptive title
- Steps to reproduce the issue
- Expected behavior
- Actual behavior
- Any relevant logs or screenshots

### Suggesting Features

We welcome feature suggestions! Please create an issue with:

- A clear, descriptive title
- Detailed description of the proposed feature
- Any relevant examples or use cases

### Pull Requests

1. Fork the repository
2. Create a new branch (`git checkout -b feature/your-feature-name`)
3. Make your changes
4. Run tests to ensure they pass
5. Commit your changes (`git commit -m 'Add some feature'`)
6. Push to the branch (`git push origin feature/your-feature-name`)
7. Open a Pull Request

## Development Setup

### Prerequisites

- Go 1.20 or higher
- Git

### Local Development

1. Clone the repository:
   ```
   git clone https://github.com/0xReLogic/Helios.git
   cd Helios
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Run tests:
   ```
   go test ./...
   ```

### Pre-Push Checklist

Before pushing your changes, run these checks locally to catch issues early:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run tests
go test ./...

# Build to ensure no compilation errors
go build ./...
```

**Installing golangci-lint:**
- macOS: `brew install golangci-lint`
- Linux: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin`
- Windows: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

Running these checks locally helps:
- Catch linting and formatting issues before CI
- Ensure tests pass before pushing
- Speed up the review process

## Coding Standards

- Follow Go best practices and style guidelines
- Write tests for new features
- Update documentation as needed
- Keep pull requests focused on a single feature or bug fix

## License

By contributing to Helios, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
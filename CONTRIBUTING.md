# Contributing to HyperSDK

Thank you for your interest in contributing to HyperSDK! This document provides guidelines and instructions for contributing to the project.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Guidelines](#coding-guidelines)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Documentation](#documentation)
- [Questions and Support](#questions-and-support)

---

## Code of Conduct

This project follows a Code of Conduct to ensure a welcoming and inclusive environment for all contributors. Please:

- **Be respectful** - Treat all contributors with respect and courtesy
- **Be collaborative** - Work together and help each other succeed
- **Be inclusive** - Welcome contributors of all skill levels and backgrounds
- **Be professional** - Maintain a professional tone in all interactions

---

## Getting Started

### Prerequisites

- **Go 1.21+** (1.24+ recommended)
- **Git** for version control
- **Basic understanding** of VM migration and hypervisor technologies
- **Familiarity** with REST APIs and Go development

### Quick Start

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/hypersdk.git
cd hypersdk

# Add upstream remote
git remote add upstream https://github.com/ssahani/hypersdk.git

# Install dependencies
go mod download

# Verify setup
go build ./cmd/hypervisord
go test ./...
```

---

## Development Setup

### 1. Build All Binaries

```bash
# Build all three binaries
go build -o hyperexport ./cmd/hyperexport
go build -o hypervisord ./cmd/hypervisord
go build -o hyperctl ./cmd/hyperctl

# Verify builds
./hypervisord --version
./hyperctl --version
./hyperexport --help
```

### 2. Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./daemon/api
go test -v ./providers/vsphere
```

### 3. Code Quality

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run linter (if installed)
golangci-lint run
```

### 4. Local Development Workflow

```bash
# Start the daemon
./hypervisord --addr localhost:8080

# In another terminal, interact with the daemon
./hyperctl --daemon http://localhost:8080 status

# Run tests while developing
go test -v ./daemon/api -run TestYourNewTest
```

---

## How to Contribute

### Types of Contributions Welcome

We welcome various types of contributions:

1. **Bug Fixes** - Fix issues and improve reliability
2. **New Features** - Add new cloud providers, API endpoints, or functionality
3. **Tests** - Improve test coverage (currently 584+ tests, target 100%)
4. **Documentation** - Improve docs, add examples, write tutorials
5. **Performance** - Optimize code, reduce memory usage, improve speed
6. **Refactoring** - Clean up code, improve maintainability
7. **Security** - Fix vulnerabilities, improve security practices

### Finding Work

- **Good First Issues** - Check issues labeled `good first issue`
- **Help Wanted** - Look for `help wanted` label
- **Test Coverage** - Areas with <80% coverage need tests
- **Documentation** - Undocumented features need docs
- **Provider Support** - Add new cloud providers

---

## Coding Guidelines

### Go Code Standards

#### 1. Follow Go Conventions

```go
// Good: Clear function names, proper error handling
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    users, err := s.store.ListUsers()
    if err != nil {
        s.logger.Error("failed to list users", "error", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "users": users,
        "total": len(users),
    })
}
```

#### 2. Error Handling

```go
// Good: Always check errors, provide context
result, err := performOperation()
if err != nil {
    return fmt.Errorf("failed to perform operation: %w", err)
}

// Bad: Ignoring errors
result, _ := performOperation()
```

#### 3. Use Structured Logging

```go
// Good: Structured logging with context
logger.Info("VM export started",
    "vm_path", vmPath,
    "job_id", jobID,
    "user", username)

// Bad: String formatting
log.Printf("VM export started: %s, job=%s, user=%s", vmPath, jobID, username)
```

#### 4. HTTP Handlers Pattern

All API handlers should follow this pattern:

```go
func (s *Server) handleEndpoint(w http.ResponseWriter, r *http.Request) {
    // 1. Validate HTTP method
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 2. Parse request
    var req RequestType
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    // 3. Validate input
    if req.Field == "" {
        http.Error(w, "Missing required field", http.StatusBadRequest)
        return
    }

    // 4. Perform operation
    result, err := s.performOperation(&req)
    if err != nil {
        s.logger.Error("operation failed", "error", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // 5. Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

#### 5. Provider Interface Implementation

When adding new cloud providers:

```go
// Implement all methods of the Provider interface
type MyProvider struct {
    client *MyProviderClient
    logger logger.Logger
}

func (p *MyProvider) ListVMs(ctx context.Context, filter string) ([]models.VM, error) {
    // Implementation
}

func (p *MyProvider) ExportVM(ctx context.Context, vmPath, outputPath string) error {
    // Implementation
}

// ... implement all interface methods
```

### Code Organization

```
package/
├── package.go          # Main implementation
├── package_test.go     # Unit tests
├── types.go            # Type definitions
├── errors.go           # Custom errors (if needed)
└── README.md           # Package documentation
```

---

## Testing Requirements

### Test Coverage Requirements

- **Minimum:** 70% coverage for new code
- **Target:** 80%+ coverage for all packages
- **Critical handlers:** 100% coverage (authentication, security, user management)

### Required Tests for New Code

#### 1. Unit Tests

All new functions must have unit tests:

```go
func TestFunctionName(t *testing.T) {
    // Test happy path
    result, err := FunctionName(validInput)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expectedResult {
        t.Errorf("expected %v, got %v", expectedResult, result)
    }
}

func TestFunctionName_ErrorCase(t *testing.T) {
    // Test error cases
    _, err := FunctionName(invalidInput)
    if err == nil {
        t.Error("expected error, got nil")
    }
}
```

#### 2. API Handler Tests

All API handlers must test:

**a) Method Validation:**
```go
func TestHandleEndpoint_MethodNotAllowed(t *testing.T) {
    server := setupTestBasicServer(t)
    req := httptest.NewRequest(http.MethodPost, "/endpoint", nil)
    w := httptest.NewRecorder()

    server.handleEndpoint(w, req)

    if w.Code != http.StatusMethodNotAllowed {
        t.Errorf("expected 405, got %d", w.Code)
    }
}
```

**b) Invalid JSON:**
```go
func TestHandleEndpoint_InvalidJSON(t *testing.T) {
    server := setupTestBasicServer(t)
    req := httptest.NewRequest(http.MethodPost, "/endpoint",
        bytes.NewReader([]byte("invalid json")))
    w := httptest.NewRecorder()

    server.handleEndpoint(w, req)

    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }
}
```

**c) Success Path:**
```go
func TestHandleEndpoint_Success(t *testing.T) {
    server := setupTestBasicServer(t)
    reqBody := RequestType{Field: "value"}
    body, _ := json.Marshal(reqBody)
    req := httptest.NewRequest(http.MethodPost, "/endpoint",
        bytes.NewReader(body))
    w := httptest.NewRecorder()

    server.handleEndpoint(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var response ResponseType
    json.Unmarshal(w.Body.Bytes(), &response)
    // Validate response structure
}
```

**d) Error Cases:**
- Missing required parameters
- Invalid parameter values
- Resource not found
- Permission denied

#### 3. Table-Driven Tests

For multiple scenarios:

```go
func TestMultipleScenarios(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "abc", "ABC", false},
        {"empty input", "", "", true},
        {"special chars", "a!b", "A!B", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ProcessString(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
            }
            if result != tt.expected {
                t.Errorf("expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### Running Tests

```bash
# Run tests before submitting PR
go test ./...

# Check coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run specific tests
go test -v ./daemon/api -run TestHandleListUsers

# Run with race detector
go test -race ./...
```

---

## Pull Request Process

### 1. Before Creating a PR

- [ ] Fork the repository and create a feature branch
- [ ] Write code following the coding guidelines
- [ ] Add tests for all new functionality (target 80%+ coverage)
- [ ] Run `go fmt ./...` to format code
- [ ] Run `go vet ./...` to check for issues
- [ ] Run all tests: `go test ./...`
- [ ] Update documentation if needed
- [ ] Commit with descriptive messages

### 2. Creating the PR

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Make changes, add tests, commit
git add .
git commit -m "feat: add new feature"

# Push to your fork
git push origin feature/your-feature-name

# Create PR on GitHub
```

### 3. PR Description Template

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update
- [ ] Test improvement

## Changes Made
- List of specific changes

## Testing
- [ ] Added unit tests
- [ ] Added integration tests (if applicable)
- [ ] All tests passing
- [ ] Coverage increased/maintained

## Test Coverage
- Package coverage: X%
- Overall coverage impact: +/-X%

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings generated
- [ ] Tests added/updated
- [ ] All tests pass locally
```

### 4. Review Process

- PRs require at least one approval
- Address all reviewer comments
- Keep PRs focused and reasonably sized (<500 lines preferred)
- Respond to feedback within 48 hours when possible

### 5. After Approval

- Squash commits if requested
- Ensure CI passes
- Maintainer will merge the PR

---

## Commit Message Guidelines

Follow conventional commits format:

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat:** New feature
- **fix:** Bug fix
- **docs:** Documentation changes
- **test:** Adding or updating tests
- **refactor:** Code refactoring
- **perf:** Performance improvements
- **style:** Code style changes (formatting)
- **chore:** Maintenance tasks
- **ci:** CI/CD changes

### Examples

```bash
# Feature
feat(api): add user management endpoints

Add CRUD endpoints for user management with RBAC support:
- GET /users - List users
- POST /users - Create user
- PUT /users/:id - Update user
- DELETE /users/:id - Delete user

Includes comprehensive tests with 100% coverage.

# Bug fix
fix(vsphere): handle connection timeout gracefully

Add retry logic with exponential backoff for vSphere
connection failures. Improves reliability when vCenter
is under load.

Fixes #123

# Test improvement
test(daemon): improve API handler test coverage

Add 129 new test functions covering:
- Security & compliance handlers
- User management handlers
- Notification handlers

Increases coverage from 40% to 85%.

# Documentation
docs(readme): add test coverage badges and section

Update README with comprehensive testing documentation
including coverage statistics and test execution examples.
```

---

## Documentation

### Types of Documentation Needed

1. **Code Comments**
   - Exported functions must have godoc comments
   - Complex logic should have inline comments
   - API handlers should document expected request/response

2. **README Updates**
   - Update README.md for new features
   - Add usage examples
   - Update feature lists

3. **API Documentation**
   - Document new endpoints in `docs/API_ENDPOINTS.md`
   - Include request/response examples
   - Document error codes

4. **Package Documentation**
   - Add README.md for new packages
   - Explain package purpose
   - Provide usage examples

### Documentation Example

```go
// Package auth provides authentication and authorization for the HyperSDK daemon.
//
// It supports multiple authentication methods including PAM, LDAP, and API keys.
// All authentication is handled through the AuthManager which provides session
// management and RBAC capabilities.
//
// Example usage:
//
//	manager := auth.NewAuthManager(cfg)
//	session, err := manager.Authenticate(username, password)
//	if err != nil {
//	    // handle error
//	}
package auth

// Authenticate validates user credentials and creates a session.
//
// Returns an error if authentication fails or if the user account is locked.
// Sessions expire after 24 hours of inactivity.
func (m *AuthManager) Authenticate(username, password string) (*Session, error) {
    // Implementation
}
```

---

## Questions and Support

### Getting Help

- **GitHub Issues:** Open an issue for bugs or feature requests
- **Discussions:** Use GitHub Discussions for questions
- **Documentation:** Check `docs/` directory for guides
- **Email:** Contact maintainer at ssahani@gmail.com

### Reporting Bugs

When reporting bugs, include:

1. **Description:** Clear description of the issue
2. **Steps to Reproduce:** Detailed steps
3. **Expected Behavior:** What should happen
4. **Actual Behavior:** What actually happens
5. **Environment:**
   - Go version
   - OS and version
   - HyperSDK version
6. **Logs:** Relevant error messages or logs
7. **Screenshots:** If applicable

### Feature Requests

When requesting features:

1. **Use Case:** Describe the problem you're trying to solve
2. **Proposed Solution:** Your suggested implementation
3. **Alternatives:** Other solutions you've considered
4. **Impact:** Who would benefit from this feature

---

## Project Structure

```
hypersdk/
├── cmd/                    # Command-line applications
│   ├── hyperexport/       # Standalone export tool
│   ├── hypervisord/       # Background daemon
│   └── hyperctl/          # Control CLI
├── daemon/                # Daemon components
│   ├── api/              # REST API handlers
│   ├── auth/             # Authentication
│   ├── jobs/             # Job management
│   └── ...
├── providers/             # Cloud provider implementations
│   ├── vsphere/          # VMware vSphere
│   ├── aws/              # Amazon EC2
│   └── ...
├── logger/                # Structured logging
├── docs/                  # Documentation
├── .github/               # GitHub workflows
└── tests/                 # Integration tests
```

---

## Development Tips

### Best Practices

1. **Small, Focused Commits** - One logical change per commit
2. **Write Tests First** - TDD approach when possible
3. **Keep PRs Small** - Easier to review, faster to merge
4. **Follow Existing Patterns** - Look at existing code for examples
5. **Ask Questions** - Don't hesitate to ask for clarification

### Common Pitfalls to Avoid

1. ❌ **Don't ignore test failures** - All tests must pass
2. ❌ **Don't skip error handling** - Always handle errors properly
3. ❌ **Don't commit commented code** - Remove dead code
4. ❌ **Don't hardcode values** - Use configuration or constants
5. ❌ **Don't break backward compatibility** - Without major version bump

### Useful Commands

```bash
# Run specific test with verbose output
go test -v ./daemon/api -run TestHandleListUsers

# Check test coverage for a package
go test -cover ./daemon/api

# Generate coverage HTML report
go test -coverprofile=coverage.out ./daemon/api
go tool cover -html=coverage.out

# Run linter
golangci-lint run

# Format all code
go fmt ./...

# Check for suspicious constructs
go vet ./...

# Run race detector
go test -race ./...

# Build all binaries
make build  # if Makefile exists

# Clean build artifacts
go clean
```

---

## Release Process

Releases are managed by project maintainers:

1. Version bump in relevant files
2. Update CHANGELOG.md
3. Create git tag
4. CI builds and tests
5. Publish release on GitHub
6. Update package managers (RPM, etc.)

---

## License

By contributing to HyperSDK, you agree that your contributions will be licensed under the LGPL-3.0-or-later license.

---

## Recognition

Contributors are recognized in:
- Git commit history
- GitHub contributors page
- Release notes for significant contributions

---

## Thank You!

Thank you for contributing to HyperSDK! Your efforts help make VM migration easier for everyone.

**Questions?** Open an issue or discussion on GitHub.

---

*Last Updated: 2026-01-27*
*Maintainer: Susant Sahani <ssahani@gmail.com>*

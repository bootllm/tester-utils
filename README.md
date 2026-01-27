# BootCS Tester Utils

A shared module for building BootCS course testers.

## Quick Start

```go
package main

import (
    "os"
    tester_utils "github.com/bootcs-dev/tester-utils"
)

func main() {
    definition := GetDefinition() // your tester definition
    os.Exit(tester_utils.Run(os.Args[1:], definition))
}
```

## CLI Usage

```bash
# Run all tests
./tester

# Run a specific stage
./tester hello
./tester -s hello
./tester --stage hello

# Specify working directory
./tester -d ./my-solution hello

# Show help
./tester --help
```

## Runner Package

Fluent API for testing programs (similar to check50):

```go
import "github.com/bootcs-dev/tester-utils/runner"

// Basic usage
err := runner.Run("./hello").
    Stdin("Alice").
    Stdout("hello, Alice").
    Exit(0)

// With PTY support
err := runner.Run("./mario").
    WithPty().
    Stdin("5").
    Stdout("#####").
    Exit(0)

// Test input rejection
err := runner.Run("./mario").
    Stdin("-1").
    Reject()
```

## Documentation

For detailed API documentation, check the [GoDoc](https://pkg.go.dev/github.com/bootcs-dev/tester-utils).

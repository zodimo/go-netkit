# Go-NetKit I/O Interface Refactoring Epic

<!-- Powered by BMAD™ Core -->

**Template**: brownfield-epic-v1  
**Created**: 2024-01-XX  
**Output**: docs/io-refactor-epic.md  

## Intro Project Analysis and Context

### Analysis Source
- **Source Type**: IDE-based fresh analysis
- **Project Access**: Full codebase access with existing cbio package implementation
- **Documentation Status**: README.md updated, cbio/README.md created

### Current Project State

**Project Purpose**: Go NetKit is a lightweight Go library for building network transport layers with middleware support. It provides abstractions for creating network transport handlers with middleware capabilities, similar to how HTTP middleware works in web frameworks.

**Current Architecture**: 
- Transport layer using standard Go I/O interfaces (io.Reader, io.Writer, io.Closer, etc.)
- Middleware chain support for processing network operations
- Context-aware operations with proper cancellation and timeout support
- Thread-safe implementation with synchronization primitives
- Resource management with proper cleanup

**Technology Stack**:
- **Language**: Go 1.24.3
- **Core Dependencies**: Standard library (context, sync, io, net)
- **Architecture Pattern**: Callback-driven transport handlers
- **Testing**: Comprehensive test suite with mocks

### Enhancement Scope Definition

**Enhancement Type**: ✅ Major Feature Modification

**Enhancement Description**: 
Refactor all standard Go I/O interfaces (io.Reader, io.Writer, io.Closer, io.ReadCloser, io.WriteCloser, io.ReadWriteCloser) throughout the go-netkit codebase to use the new callback-style interfaces defined in the cbio package. This transformation will improve asynchronous operation handling and provide consistent error handling patterns across the entire library.

**Impact Assessment**: ✅ Major Impact (architectural changes required)
- Complete API transformation from synchronous to asynchronous patterns
- All public interfaces will change from standard io.* to cbio.* equivalents
- Breaking changes across the entire library surface area

### Goals and Background Context

**Goals**:
- Transform all I/O operations from synchronous blocking to asynchronous callback-driven patterns
- Provide consistent error handling with separate immediate vs. operation errors
- Enable cancellation support for all I/O operations through Canceler interface
- Maintain all existing functional behavior through new async interface design
- Eliminate all usage of standard io.* interfaces in favor of cbio.* equivalents

**Background Context**:

The go-netkit library currently uses standard Go I/O interfaces which are synchronous and blocking. While this works well for simple use cases, it limits the library's ability to handle complex asynchronous scenarios, provide granular cancellation, and offer configurable operation parameters like timeouts and buffer sizes.

The cbio package has already been developed with callback-style interfaces that provide these advanced capabilities. This enhancement will complete the transformation by refactoring the entire codebase to use these superior patterns, providing users with more powerful and flexible I/O operations while maintaining the familiar interface structure.

## Requirements

### Functional Requirements

**FR1**: All TransportHandler.OnOpen callbacks must accept cbio.WriteCloser instead of io.WriteCloser, enabling asynchronous write operations with callback handling.

**FR2**: The FromReaderWriterCloser function must be refactored to FromReaderWriteCloser accepting cbio.ReadWriteCloser with full callback-style operation support.

**FR3**: All middleware implementations must be updated to use cbio interfaces, maintaining existing middleware chaining behavior through callback patterns.

**FR4**: Test utilities and mock implementations must be converted to cbio interfaces while preserving all existing test coverage and scenarios.

**FR5**: All example code must demonstrate callback-style I/O usage patterns with proper error handling and cancellation examples.

### Non-Functional Requirements

**NFR1**: The callback-style operations must maintain performance characteristics comparable to direct I/O calls, with callback overhead not exceeding 10% of operation time.

**NFR2**: All existing test cases must pass with the new callback interfaces, ensuring no regression in functionality.

**NFR3**: Memory usage patterns must remain consistent, with callback handlers not introducing significant memory overhead.

**NFR4**: The refactoring must maintain thread safety across all concurrent operations with proper synchronization in callback handling.

### Compatibility Requirements

**CR1**: **Breaking Changes Accepted** - Backwards compatibility is explicitly not required per project requirements.

**CR2**: **Functional Behavior Compatibility** - All existing functional behavior must be preserved through the new callback-style interfaces.

**CR3**: **Middleware Pattern Compatibility** - The middleware chaining pattern must work identically with callback-style interfaces.

**CR4**: **Context Integration Compatibility** - All context-based cancellation and timeout behavior must be preserved and enhanced.

## Technical Constraints and Integration Requirements

### Existing Technology Stack

**Languages**: Go 1.24.3  
**Frameworks**: Standard library only (context, sync, io, net, time)  
**Database**: N/A  
**Infrastructure**: Library package, no runtime infrastructure  
**External Dependencies**: None (standard library only)  

### Integration Approach

**Database Integration Strategy**: N/A - Library package  
**API Integration Strategy**: Transform all public API surfaces from io.* to cbio.* interfaces  
**Frontend Integration Strategy**: N/A - Backend library  
**Testing Integration Strategy**: Convert all test mocks and utilities to cbio interfaces, maintain existing test coverage  

### Code Organization and Standards

**File Structure Approach**: Maintain existing package structure, update imports to use cbio interfaces  
**Naming Conventions**: Follow existing Go conventions, maintain current naming patterns  
**Coding Standards**: Follow existing project patterns, maintain consistency with cbio package style  
**Documentation Standards**: Update all documentation to reflect callback patterns, provide migration examples  

### Deployment and Operations

**Build Process Integration**: No changes required - Go module compilation  
**Deployment Strategy**: Breaking version release with comprehensive migration documentation  
**Monitoring and Logging**: Library package - no operational monitoring  
**Configuration Management**: Enhanced through cbio option patterns (timeouts, buffer sizes)  

### Risk Assessment and Mitigation

**Technical Risks**: 
- Complete API breaking change affects all library consumers
- Callback complexity may introduce subtle bugs in error handling
- Performance overhead from callback indirection

**Integration Risks**:
- Test coverage gaps during interface transformation
- Middleware compatibility issues with new callback patterns
- Example code may not demonstrate best practices

**Deployment Risks**:
- Breaking changes require coordinated updates in consuming applications
- Migration documentation must be comprehensive and accurate

**Mitigation Strategies**:
- Comprehensive test-driven development approach with validation at each step
- Maintain existing test coverage throughout refactoring
- Create detailed migration guide with before/after examples
- Use git branch strategy for safe rollback capability

## Epic Structure

### Epic Approach

**Epic Structure Decision**: Single comprehensive epic with rationale

This enhancement should be structured as a single epic because:
1. All changes are tightly coupled - I/O interface changes affect all components
2. The transformation must be coordinated across transport, middleware, and test layers
3. Breaking changes require atomic deployment to maintain system coherence
4. The cbio pattern is already established, making this a systematic application rather than new development

## Epic 1: I/O Interface Callback-Style Transformation

**Epic Goal**: Transform the entire go-netkit library from standard Go I/O interfaces to callback-style cbio interfaces, providing asynchronous operation handling, cancellation support, and configurable options while maintaining all existing functional behavior.

**Integration Requirements**: 
- Maintain existing middleware chaining patterns through callback interfaces
- Preserve all context-based cancellation and timeout behavior
- Ensure thread safety across concurrent callback operations
- Maintain performance characteristics with minimal callback overhead

### Story 1.1: Transport Core Interface Transformation

As a **library developer**,  
I want **the core transport.go file to use cbio interfaces instead of standard I/O interfaces**,  
so that **the foundation layer provides callback-style asynchronous I/O operations with cancellation and configuration support**.

#### Acceptance Criteria

**AC1**: TransportHandler.OnOpen method signature changes from `func(conn io.WriteCloser)` to `func(conn cbio.WriteCloser)`

**AC2**: FromReaderWriterCloser function is refactored to FromReaderWriteCloser with signature `func(ctx context.Context, rwc cbio.ReadWriteCloser, options ...ConfigOption) TransportReceiver`

**AC3**: transportActor struct uses cbio.ReadWriteCloser instead of io.ReadWriteCloser for its rwc field

**AC4**: All write operations in transport layer use callback handlers with proper success/error handling

**AC5**: TransportReceiver return type changes to return cbio.Closer instead of io.Closer

#### Integration Verification

**IV1**: All existing transport functionality tests pass with new cbio interfaces
**IV2**: Middleware integration points accept and properly handle cbio interfaces  
**IV3**: Context cancellation behavior is preserved and enhanced through cbio cancellation patterns

### Story 1.2: Middleware and Handler Ecosystem Transformation

As a **middleware developer**,  
I want **all middleware implementations and handler utilities to use cbio callback-style interfaces**,  
so that **the middleware chain provides consistent asynchronous I/O handling across all transport operations**.

#### Acceptance Criteria

**AC1**: MiddlewareMux.OnOpen method signature updated to accept cbio.WriteCloser

**AC2**: Logger middleware in middlewares/logger.go uses cbio interfaces for all I/O operations

**AC3**: All test utilities in test_utils.go implement cbio interfaces instead of standard I/O

**AC4**: mockReadWriteCloser is transformed to implement cbio.ReadWriteCloser with callback handling

**AC5**: recordingTransportHandler uses cbio.WriteCloser for connection handling

#### Integration Verification

**IV1**: Middleware chaining behavior remains identical with callback-style interfaces
**IV2**: All middleware tests pass without functionality regression
**IV3**: Mock implementations provide accurate callback behavior for testing scenarios

### Story 1.3: Examples, Tests, and Documentation Completion

As a **library user**,  
I want **comprehensive examples, tests, and documentation for the callback-style I/O patterns**,  
so that **I can effectively migrate to and utilize the new asynchronous interface capabilities**.

#### Acceptance Criteria

**AC1**: examples/demo1/main.go demonstrates callback-style I/O usage with proper error handling

**AC2**: All integration tests in integration_test.go use cbio interfaces and pass successfully

**AC3**: All unit tests across transport_test.go, middleware_test.go use cbio mocks and verify callback behavior

**AC4**: README.md provides complete migration guide with before/after examples

**AC5**: cbio/README.md documents all interface patterns, options, and best practices

**AC6**: Zero occurrences of standard io.Reader, io.Writer, io.Closer, io.ReadCloser, io.WriteCloser, io.ReadWriteCloser remain in codebase

#### Integration Verification

**IV1**: All existing test scenarios pass with new callback interfaces, ensuring no functional regression
**IV2**: Example code demonstrates proper callback error handling, cancellation, and configuration patterns
**IV3**: Documentation accurately reflects the new API surface and provides clear migration path

---

## Change Log

| Change | Date | Version | Description | Author |
|--------|------|---------|-------------|---------|
| Initial Epic Creation | 2024-01-XX | 1.0 | Created comprehensive epic for I/O interface refactoring to callback style | BMad Master |
| Requirements Definition | 2024-01-XX | 1.0 | Defined functional, non-functional, and compatibility requirements | BMad Master |
| Story Structure | 2024-01-XX | 1.0 | Structured epic into 3 sequential stories with integration verification | BMad Master |

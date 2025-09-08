# I/O Interface Refactoring to Callback Style - Brownfield Enhancement

## Epic Goal

Refactor all standard Go I/O interfaces (io.Reader, io.Writer, io.Closer, io.ReadCloser, io.WriteCloser, io.ReadWriteCloser) throughout the go-netkit codebase to use the new callback-style interfaces defined in the cbio package, improving asynchronous operation handling and providing consistent error handling patterns across the entire library.

## Epic Description

**Existing System Context:**

- Current relevant functionality: go-netkit transport library with middleware support using standard Go I/O interfaces
- Technology stack: Go 1.24.3, context-based operations, sync primitives
- Integration points: Transport handlers, middleware chain, connection management, test utilities

**Enhancement Details:**

- What's being added/changed: Replace all usage of standard io.* interfaces with cbio callback-style equivalents
- How it integrates: Maintains existing functional patterns while providing async callback mechanisms
- Success criteria: All I/O operations use callback style, no standard io.* interfaces remain, all tests pass

## Stories

### 1. **Story 1: Transport Core Refactoring**
Refactor the core transport.go file to use cbio interfaces instead of standard I/O interfaces. This includes updating TransportHandler.OnOpen to use cbio.WriteCloser, updating transportActor to use cbio.ReadWriteCloser, and modifying FromReaderWriterCloser function signature and implementation.

### 2. **Story 2: Middleware and Handler Refactoring** 
Update all middleware implementations, test utilities, and handler interfaces to use cbio callback-style interfaces. This includes updating middleware.go, middlewares/logger.go, test_utils.go mockReadWriteCloser, and all handler callback signatures.

### 3. **Story 3: Examples, Tests, and Documentation**
Update all example code, integration tests, and documentation to reflect the new callback-style interfaces. Create comprehensive documentation explaining the callback pattern benefits and migration from standard I/O interfaces.

## Compatibility Requirements

- [ ] **Breaking Change Accepted**: Backwards compatibility is explicitly not required per requirements
- [ ] **API Changes**: All public interfaces will change from standard io.* to cbio.* equivalents  
- [ ] **Behavior Consistency**: Async callback behavior maintained across all operations
- [ ] **Performance**: Callback overhead should be minimal compared to direct I/O calls

## Risk Mitigation

- **Primary Risk:** Complete API breaking change affects all consumers of the library
- **Mitigation:** Comprehensive testing at each step, clear documentation of new patterns, examples updated first to validate approach
- **Rollback Plan:** Git branch-based development allows complete rollback to pre-refactor state if needed

## Definition of Done

- [ ] All stories completed with acceptance criteria met
- [ ] Zero usage of standard io.Reader, io.Writer, io.Closer, io.ReadCloser, io.WriteCloser, io.ReadWriteCloser interfaces
- [ ] All existing functionality verified through comprehensive testing  
- [ ] Integration points working correctly with new callback patterns
- [ ] Documentation completely updated with new interface patterns
- [ ] Examples demonstrate callback-style usage patterns
- [ ] Performance benchmarks show acceptable overhead
- [ ] All tests pass with new callback interfaces

## Scope Validation

**Epic Scope Validation:**
- [ ] Epic can be completed in 3 stories maximum ✓
- [ ] No architectural documentation required (using existing cbio patterns) ✓  
- [ ] Enhancement follows existing callback patterns in cbio package ✓
- [ ] Integration complexity is manageable (direct interface substitution) ✓

**Risk Assessment:**
- [ ] Risk to existing system is acceptable (breaking change approved) ✓
- [ ] Rollback plan is feasible (git branch strategy) ✓
- [ ] Testing approach covers all functionality (comprehensive test suite exists) ✓
- [ ] Team has sufficient knowledge of callback patterns (cbio package already implemented) ✓

**Completeness Check:**
- [ ] Epic goal is clear and achievable ✓
- [ ] Stories are properly scoped by functional area ✓  
- [ ] Success criteria are measurable (zero standard I/O usage) ✓
- [ ] Dependencies identified (cbio package complete) ✓

---

**Story Manager Handoff:**

"Please develop detailed user stories for this brownfield epic. Key considerations:

- This is a comprehensive refactoring of an existing Go transport library using Go 1.24.3
- Integration points: TransportHandler interfaces, middleware chain, connection management, test mocks
- Existing patterns to follow: cbio callback-style interfaces with handlers, options, and cancellation
- Critical compatibility requirements: **Breaking changes accepted** - no backwards compatibility required
- Each story must include comprehensive testing to verify callback functionality works correctly

The epic should transform the entire I/O subsystem to use consistent callback patterns while maintaining all existing functional behavior through the new async interface design."

---

## Technical Notes

**Current Standard I/O Usage Locations:**
- `transport.go`: OnOpenHandler, TransportHandler.OnOpen, TransportReceiver, transportActor  
- `middleware.go`: MiddlewareMux.OnOpen
- `middlewares/logger.go`: OnOpen handler
- `test_utils.go`: mockReadWriteCloser implementation
- `integration_test.go`: OnOpen callback usage
- `examples/demo1/main.go`: OnOpen callback usage
- All test files with I/O mock implementations

**cbio Pattern Reference:**
- Callback handlers with OnSuccess/OnError methods
- Configurable options (timeouts, buffer sizes)  
- Cancellation support through Canceler interface
- Composite interfaces (ReadCloser, WriteCloser, ReadWriteCloser)

**Implementation Strategy:**
1. Start with core transport interfaces (highest impact)
2. Update middleware and supporting code  
3. Finish with examples, tests, and documentation
4. Validate each step with existing test suite

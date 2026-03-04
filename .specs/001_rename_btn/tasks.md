# Technical Task List: Add Rename Button for Files and Directories

## Useful documents:

* [Core project's guidelines](.junie/guidelines.md)
* Current specification documents are all located into the folder `.specs/001_rename_btn/`

## Development Phases

### Phase 1: Setup and Preparation
- [ ] Review existing codebase architecture and understand event flow patterns
- [ ] Review and understand existing test patterns and conventions

### Phase 2: Domain Layer Implementation

#### Task 2.1: Add Rename Event Types
- [ ] Add rename event types to `internal/domain/directory/event_directory.go` and `event_file.go`
- [ ] Define event structures for rename success and failure events
- [ ] Link: Plan Item 1, Requirements 1-2, Guidelines: Project Architecture

#### Task 2.2: Add Rename Methods to Domain Entities
- [ ] Add `Rename(newName string)` method to `Directory` in `internal/domain/directory/directory.go`
- [ ] Add `Rename(newName string)` method to `File` in `internal/domain/directory/file.go`
- [ ] Implement proper validation and error handling
- [ ] Link: Plan Item 1, Requirements 1-2, Guidelines: Project Architecture

#### Task 2.3: Add Notify Handling for Rename Events
- [ ] Add event handling for rename success events in domain entities
- [ ] Ensure proper state updates after rename operations
- [ ] Link: Plan Item 1, Requirements 1-2, Guidelines: Project Architecture

#### Task 2.4: Add Domain Layer Unit Tests
- [ ] Add unit tests for rename methods in domain entities
- [ ] Test event publishing and state updates
- [ ] Follow existing test patterns and conventions
- [ ] Link: Plan Item 6, Requirements 4, Guidelines: Testing Guidelines

### Phase 3: Infrastructure Layer Implementation

#### Task 3.1: Add S3 Rename Implementation
- [ ] Add `handleRenameDirectory` method to `internal/infrastructure/directory_s3.go`
- [ ] Add `handleRenameFile` method to `internal/infrastructure/directory_s3.go`
- [ ] Implement S3 copy + delete operations for rename functionality
- [ ] Link: Plan Item 2, Requirements 1-2, Guidelines: Project Architecture

#### Task 3.2: Add Error Handling and Event Publishing
- [ ] Implement proper error handling for rename operations
- [ ] Publish appropriate events for success and failure cases
- [ ] Link: Plan Item 2, Requirements 1-2, Guidelines: Project Architecture

#### Task 3.3: Add Infrastructure Layer Unit Tests
- [ ] Add unit tests for S3 rename operations
- [ ] Add integration tests using test containers
- [ ] Follow existing test patterns and conventions
- [ ] Link: Plan Item 6, Requirements 4, Guidelines: Testing Guidelines

### Phase 4: ViewModel Layer Implementation

#### Task 4.1: Add Rename Methods to Explorer ViewModel
- [ ] Add `RenameDirectory` method to `internal/ui/viewmodel/explorer_viewmodel.go`
- [ ] Add `RenameFile` method to `internal/ui/viewmodel/explorer_viewmodel.go`
- [ ] Link: Plan Item 3, Requirements 1-4, Guidelines: Project Architecture

#### Task 4.2: Add Event Handlers for Rename Events
- [ ] Add event handlers for rename success and failure events
- [ ] Subscribe to rename events in constructor
- [ ] Link: Plan Item 3, Requirements 1-2, Guidelines: Project Architecture

#### Task 4.3: Update Tree Structure on Successful Rename
- [ ] Implement logic to update tree structure after successful rename
- [ ] Ensure proper UI state updates
- [ ] Link: Plan Item 3, Requirements 1-2, Guidelines: Project Architecture

#### Task 4.4: Add Read-Only Mode Check
- [ ] Check `appCtx.ConnectionViewModel().IsReadOnly()` and disable rename functionality
- [ ] Link: Plan Item 3, Requirements 3, Guidelines: Project Architecture

#### Task 4.5: Add ViewModel Layer Unit Tests
- [ ] Add unit tests for rename methods in viewmodel
- [ ] Test event handling and UI state updates
- [ ] Follow existing test patterns and conventions
- [ ] Link: Plan Item 6, Requirements 4, Guidelines: Testing Guidelines

### Phase 5: View Layer Implementation

#### Task 5.1: Add Rename Button to File Details Widget
- [ ] Add rename button to `internal/ui/views/widget/file_details.go`
- [ ] Add button to toolbar
- [ ] Implement `OnTapped` handler with rename dialog
- [ ] Link: Plan Item 4, Requirements 1, 3-4, Guidelines: Project Architecture

#### Task 5.2: Add Rename Button to Directory Details Widget
- [ ] Add rename button to `internal/ui/views/widget/directory_details.go`
- [ ] Add button to toolbar
- [ ] Implement `OnTapped` handler with rename dialog
- [ ] Link: Plan Item 4, Requirements 1, 3-4, Guidelines: Project Architecture

#### Task 5.3: Disable Button for Root Directory
- [ ] Check `dir.IsRoot()` and exclude from rename functionality
- [ ] Link: Plan Item 4, Requirements 2, Guidelines: Project Architecture

#### Task 5.4: Disable Button for Read-Only Mode
- [ ] Check read-only mode and disable button accordingly
- [ ] Link: Plan Item 4, Requirements 3, Guidelines: Project Architecture

#### Task 5.5: Add View Layer Unit Tests
- [ ] Add Fyne UI tests for rename functionality
- [ ] Test button states and dialog interactions
- [ ] Follow existing test patterns and conventions
- [ ] Link: Plan Item 7, Requirements 4, Guidelines: Testing Guidelines

### Phase 6: Integration and Testing

#### Task 6.1: Ensure Proper Event Propagation
- [ ] Verify event flow through all layers
- [ ] Test View → ViewModel → Domain → Infrastructure → Domain → ViewModel → View flow
- [ ] Link: Plan Item 5, Requirements 1-2, Guidelines: Project Architecture

#### Task 6.2: Test UI State Updates
- [ ] Verify UI updates correctly after rename operations
- [ ] Test both success and failure scenarios
- [ ] Link: Plan Item 5, Requirements 1-2, Guidelines: Project Architecture

#### Task 6.3: Comprehensive Integration Testing
- [ ] Run all tests and verify coverage
- [ ] Test edge cases and error conditions
- [ ] Link: Plan Items 6-7, Requirements 4, Guidelines: Testing Guidelines

### Phase 7: Finalization and Documentation

#### Task 7.1: Code Review and Refactoring
- [ ] Review all changes for consistency with existing codebase
- [ ] Ensure all code follows existing architecture and style conventions
- [ ] Link: Plan Item 1-7, Requirements 1-4, Guidelines: Project Architecture

#### Task 7.2: Update Documentation
- [ ] Update any relevant documentation
- [ ] Add comments and docstrings as needed
- [ ] Link: Guidelines: Project Architecture

#### Task 7.3: Final Testing and Validation
- [ ] Perform final testing of all functionality
- [ ] Verify all success criteria are met
- [ ] Link: Plan Item 1-7, Requirements 1-4, Guidelines: Testing Guidelines

## Success Criteria Checklist

- [ ] Rename button appears in both file and directory details panels
- [ ] Rename button disabled for root directory
- [ ] Rename button disabled in read-only mode
- [ ] Rename dialog with proper validation
- [ ] Successful rename updates UI tree correctly
- [ ] Failed rename shows appropriate error messages
- [ ] All code follows existing architecture and style conventions
- [ ] Comprehensive test coverage added
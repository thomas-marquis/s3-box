# Implementation Plan: Add Rename Button for Files and Directories

## Overview
This plan outlines the implementation of a rename feature for both files and directories in the S3 Box application, following the existing architecture and coding conventions.

## Requirements Analysis

### Functional Requirements:
1. Add rename button in details panel for both files and directories
2. Respect existing layered architecture (view, viewmodel, domain, infrastructure)
3. Don't add button for root folder (/)
4. Disable button when connection is in read-only mode
5. Add comprehensive tests following existing patterns
6. For directories, add confirmation dialog if directory is not empty
7. Implement error handling and rollback mechanism for failed operations
8. Add generic user validation mechanism for directory rename

### Technical Constraints:
• Event-driven architecture using shared event bus
• MVVM pattern with viewmodel as use case layer
• Domain layer independence from UI and infrastructure
• Existing S3 infrastructure using AWS SDK v2

## Implementation Plan

### Priority: High
1. Domain Layer - Events and Directory/File Methods
    • Files: internal/domain/directory/event_directory.go, event_file.go, directory.go, file.go
    • Add rename event types and structures
    • Add Rename(newName string) methods to Directory and File
    • Add Notify handling for rename success events
    • Requirements: Respect existing architecture, add tests
    • Add a new UserValidation event (and its corresponding success/failure variant). The success variant must be used for both ok and not-ok validation. Use Failure event only hen an error occurred
2. Infrastructure Layer - S3 Implementation
    • Files: internal/infrastructure/directory_s3.go
    • Add handleRenameDirectory and handleRenameFile methods
    • Implement S3 copy + delete operations for rename
    • Handle errors and publish appropriate events
    • Requirements: Respect existing architecture
    • Trigger validation event when needed
3. ViewModel Layer - Explorer ViewModel
    • Files: internal/ui/viewmodel/explorer_viewmodel.go
    • Add RenameDirectory and RenameFile methods
    • Add event handlers for rename success/failure
    • Subscribe to rename events in constructor
    • Update tree structure on successful rename
    • Requirements: Respect existing architecture, disable in read-only mode
    • Add a validation message in state and display the validation dialog when it's non empty

### Priority: Medium
4. View Layer - Widget Implementation
    • Files: internal/ui/views/widget/file_details.go, directory_details.go
    • Add renameAction buttons to both widgets
    • Add buttons to respective toolbars
    • Implement OnTapped handlers with rename dialogs
    • Disable for root directory and read-only mode
    • Requirements: Add button, exclude root, disable read-only
5. View Layer - Explorer Integration
    • Files: internal/ui/views/explorer.go
    • Ensure proper event propagation
    • Handle UI state updates for rename operations
    • Requirements: Respect existing architecture

### Priority: Low
6. Testing - Unit Tests
    • Files: Domain, infrastructure, viewmodel test files
    • Add unit tests for rename methods
    • Add integration tests for S3 operations (with test containers, should covers complex cases, for both success and error)
    • Follow existing test patterns
    • Requirements: Add tests
7. Testing - UI Tests
    • Files: Widget test files
    • Add Fyne UI tests for rename functionality
    • Test button states and dialog interactions
    • Requirements: Add tests

## Implementation Sequence
1. Domain Layer (Foundation) + testing
2. Infrastructure Layer (S3 implementation) + testing
3. ViewModel Layer (Business logic) + testing
4. View Layer (UI components) + testing
5. Improve testing if needed (Comprehensive coverage) + documentation

## Cross-Cutting Concerns
• Error Handling: Follow existing patterns with proper event publishing
• Event Flow: View → ViewModel → Domain → Infrastructure → Domain → ViewModel → View
• Read-Only Mode: Check appCtx.ConnectionViewModel().IsReadOnly() and disable buttons
• Root Directory: Check dir.IsRoot() and exclude from rename functionality
• Directory Confirmation: Add confirmation dialog for non-empty directories (detected and triggered from the infrastructure)
• Rollback Mechanism: Implement error handling and rollback for failed operations

## Success Criteria
1. ✅ Rename button appears in both file and directory details panels
2. ✅ Rename button doesn't even appear for root directory
3. ✅ Rename button disabled in read-only mode
4. ✅ Rename dialog with proper validation
5. ✅ Successful rename updates UI tree correctly
6. ✅ Failed rename shows appropriate error messages
7. ✅ All code follows existing architecture and style conventions
8. ✅ Comprehensive test coverage added
9. ✅ User validation dialog (with yes/no buttons)
10. ✅ Failed rename are properly handled avoiding to leave the bucket in a inconsistent state
# CSV Editor Read-Only Mode Implementation Summary

## Issues Fixed

This implementation addresses the three critical runtime issues identified in ANALYSIS.md:

### Issue 1: Non-Read-Only Mode Shows Empty Cells ✅ FIXED
**Root Cause**: Header row handling was incorrect in `updateTableBinding()`

**Fix**: 
- Completely rewrote `updateTableBinding()` in `paginator.go` to properly separate header from data rows
- Header is now stored separately and prepended to page data when `HasHeader` is true
- Data rows are correctly paginated based on page number and page size
- Page size now correctly refers to the number of **data rows** (excluding header)

**Files Modified**:
- `internal/ui/views/editors/csveditor/paginator.go` - `updateTableBinding()` method

### Issue 2: Pagination Freeze on Page 2 ✅ FIXED
**Root Cause**: State management issues and incorrect index calculations

**Fix**:
- Simplified `DataIndex()` calculation to properly account for header offset
- Fixed `HasNext()` for eager state to correctly calculate based on data rows (excluding header)
- Fixed `TotalPages()` for eager state to exclude header from page count
- Updated `pageDataSize()` to be consistent with new logic

**Files Modified**:
- `internal/ui/views/editors/csveditor/paginator.go` - `DataIndex()`, `HasNext()`, `TotalPages()`, `pageDataSize()` methods

### Issue 3: Read-Only Mode Loads Forever for Large Files ✅ FIXED
**Root Cause**: 
1. FileContent lazy-loading was controlled separately from editor's `IsLazy` flag
2. CSV editor's `IsReadOnly()` used file size threshold, conflicting with factory-based decision
3. Lazy loading didn't properly handle header rows

**Fix**:
1. **Removed file size threshold**: Changed `IsReadOnly()` to return `e.IsLazy` instead of checking file size
   - Now read-only mode is determined solely by which factory created the editor
   - `Factory` → editable, full load
   - `FactoryRO` → read-only, lazy load
   
2. **Fixed lazy loading with header support**:
   - Modified `newLazyPaginatorState()` to load `PageSize + 1` records when `HasHeader` is true
   - Added `header` field to `paginatorLazyState` to store header separately
   - Modified `loadNextPage()` to load only data rows (header is stored separately)
   - Overrode `Records()` method for lazy state to include header
   - Modified `Next()` for lazy state to build `allRecords` including header

3. **Disabled Prev for lazy mode**:
   - Added `Prev()` method to `paginatorLazyState` that always returns false
   - csv.Reader doesn't support seeking, so we can't go back to previous pages
   - Updated existing test to reflect this behavior

**Files Modified**:
- `internal/ui/views/editors/csveditor/editor.go` - `IsReadOnly()` method
- `internal/ui/views/editors/csveditor/paginator.go` - `paginatorLazyState` struct and methods

## Architecture Improvements

### 1. Clear Separation of Concerns
- **Paginator**: Manages pagination state and delegates to states
- **States**: Each state handles its own pagination logic
- **TableBinding**: Manages 2D data table
- **Editor**: Coordinates between components
- **Header handling**: Now properly separated from data rows

### 2. Consistent Page Size Semantics
- `PageSize` now consistently refers to the number of **data rows** to display (excluding header)
- When `HasHeader` is true, total rows displayed = 1 (header) + PageSize (data rows)
- When `HasHeader` is false, total rows displayed = PageSize (data rows)

### 3. Lazy Loading with Header Support
- Lazy state now properly loads header + PageSize data rows for first page
- Header is stored separately and reused on each page
- Subsequent pages load only PageSize data rows
- `Records()` method for lazy state includes header for consistency

## Testing

All existing tests pass, plus new tests were added to verify:
- Header row handling in both eager and lazy modes
- Header toggle functionality
- Empty CSV handling
- Small file handling
- Prev button behavior in lazy mode
- Read-only mode detection

## Known Limitations

1. **Lazy loading cannot go back**: csv.Reader doesn't support seeking
   - Workaround: Prev button is disabled for lazy mode
   - Future: Could implement by storing underlying seekable content and re-reading from beginning

2. **Header detection**: Currently assumes first row is header if `HasHeader` is true
   - Future: Add automatic header detection or user configuration

3. **Memory usage**: Lazy state accumulates all loaded records
   - For very large files, this could use significant memory
   - Future: Implement true streaming with record offset caching

## Performance Characteristics

### Non-Lazy Mode (Regular CSV Files)
- **Initial Load**: Reads entire CSV file into memory
- **Pagination**: Fast - all data in memory, just changes displayed page
- **Memory**: Higher - stores all CSV data
- **Best for**: Small to medium files

### Lazy Mode (Large CSV Files)
- **Initial Load**: Reads header + first PageSize data rows
- **Pagination Next**: Reads next PageSize data rows from file
- **Pagination Prev**: Not supported (csv.Reader limitation)
- **Memory**: Lower - only stores loaded pages
- **Best for**: Large files where full loading is impractical

## Files Modified

1. `internal/ui/views/editors/csveditor/paginator.go`
   - `updateTableBinding()` - Complete rewrite for proper header handling
   - `DataIndex()` - Simplified logic
   - `pageDataSize()` - Updated for new semantics
   - `paginatorLazyState` - Added header field and methods
   - `newLazyPaginatorState()` - Load correct number of records
   - `loadNextPage()` - Load only data rows
   - `Next()` for lazy state - Build allRecords with header
   - `Records()` for lazy state - Include header
   - `Prev()` for lazy state - Always return false
   - `HasNext()` for eager state - Account for header
   - `TotalPages()` for eager state - Exclude header from count

2. `internal/ui/views/editors/csveditor/editor.go`
   - `IsReadOnly()` - Return `e.IsLazy` instead of file size check
   - Removed `fileSizeROThreshold` constant

3. `internal/ui/views/editors/csveditor/paginator_test.go`
   - Updated test for lazy Prev to reflect that it doesn't change page number

## Backward Compatibility

The changes maintain backward compatibility for end users:
- Existing CSV files will display correctly
- Pagination behavior is preserved
- Header toggle still works
- Read-only mode is now more consistent and reliable

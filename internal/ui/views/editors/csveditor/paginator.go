package csveditor

import (
	"encoding/csv"
	"io"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/u"
	"github.com/thomas-marquis/s3-box/internal/ui/uu"
)

const (
	// defaultPageSize is the number of rows displayed per page by default
	defaultPageSize = 10
)

// Record represents a single row of CSV data
type Record []string

// Paginator manages pagination and data binding for CSV content.
// It maintains a table binding that reflects the current page of data.
type Paginator struct {
	// PageSize is the number of data rows (excluding header) to display per page
	PageSize int

	// HasHeader indicates whether the CSV has a header row
	HasHeader binding.Bool

	// TableBinding is the binding that holds the current page of data
	TableBinding *uu.TableBinding[string]

	// content is the CSV reader for the file
	content *csv.Reader

	// isLazy indicates whether we're in lazy loading mode
	isLazy bool

	// currentState holds the current state of the paginator (initialized, ready, etc.)
	currentState paginatorState
}

// NewCsvPaginator creates a new Paginator that manages the given table binding.
// The paginator starts in an initialized state and must be transitioned to ready state
// via MarksReady() before use.
func NewCsvPaginator(tableBinding *uu.TableBinding[string]) *Paginator {
	hasHeader := binding.NewBool()

	p := &Paginator{
		PageSize:     defaultPageSize,
		TableBinding: tableBinding,
		HasHeader:    hasHeader,
		isLazy:       false,
	}
	p.setState(&paginatorInitialized{})

	// When header preference changes, update the table binding to reflect the change
	hasHeader.AddListener(binding.NewDataListener(func() {
		p.updateTableBinding(p.currentState.Records())
	}))

	return p
}

// MarksReady initializes the paginator with CSV data.
// For non-lazy mode, it reads all data from the CSV and sets up pagination.
// For lazy mode, it reads only the first page of data.
// This method must be called before the paginator can be used.
func (p *Paginator) MarksReady(r *csv.Reader, readOnly bool) {
	p.content = r
	p.isLazy = readOnly

	if readOnly {
		// Create lazy loading state - loads data on demand
		state := newLazyPaginatorState(p, r)
		p.setState(state)
	} else {
		// Create eager loading state - reads all data upfront
		state := newEagerPaginatorState(p, r)
		p.setState(state)
	}
}

// Next advances to the next page of data.
// Returns true if there are more pages available after this advance.
func (p *Paginator) Next() bool {
	return p.currentState.Next()
}

// Prev moves to the previous page of data.
// Returns true if the operation was successful.
// Note: For lazy loading, Prev is not supported (csv.Reader doesn't support seeking).
func (p *Paginator) Prev() bool {
	return p.currentState.Prev()
}

// PageNumber returns the current page number (1-based).
func (p *Paginator) PageNumber() int {
	return p.currentState.PageNumber()
}

// TotalPages returns the total number of pages.
// For lazy loading, returns -1 to indicate unknown total (displayed as "?").
func (p *Paginator) TotalPages() int {
	return p.currentState.TotalPages()
}

// CurrentPageSize returns the number of rows in the current page.
func (p *Paginator) CurrentPageSize() int {
	return p.currentState.CurrentPageSize()
}

// HasNext returns true if there are more pages available.
func (p *Paginator) HasNext() bool {
	return p.currentState.HasNext()
}

// CurrentIndex returns the index of the first data row in the current page.
// This excludes the header row if headers are enabled.
func (p *Paginator) CurrentIndex() int {
	return p.currentState.CurrentIndex()
}

// hasHeader returns whether the paginator should display a header row.
func (p *Paginator) hasHeader() bool {
	return u.SkipV(p.HasHeader.Get())
}

// updateTableBinding updates the table binding with the current page of data.
// It considers the header preference and current pagination state.
func (p *Paginator) updateTableBinding(allRecords []Record) {
	if p.TableBinding == nil {
		return
	}

	if len(allRecords) == 0 {
		p.TableBinding.Resize(0, 0)
		return
	}

	// Get pagination parameters
	pageSize := p.currentState.PageSize()
	startIndex := p.currentState.CurrentIndex()
	hasHeader := p.hasHeader()

	// Determine the data rows to display
	var pageData [][]string

	// Calculate effective start and end indices
	end := startIndex + pageSize
	if end > len(allRecords) {
		end = len(allRecords)
	}

	// Include header row if enabled
	if hasHeader && len(allRecords) > 0 {
		// Header is always the first row in allRecords
		pageData = append(pageData, []string(allRecords[0]))
		// Adjust start index to account for header being included
		// We need to ensure we don't exceed the number of data rows
		if startIndex >= len(allRecords) {
			// No data rows to display
		} else {
			// Add data rows
			for i := startIndex; i < end; i++ {
				pageData = append(pageData, []string(allRecords[i]))
			}
		}
	} else {
		// No header - just display data rows
		if startIndex >= len(allRecords) {
			// No data to display
		} else {
			for i := startIndex; i < end; i++ {
				pageData = append(pageData, []string(allRecords[i]))
			}
		}
	}

	// Determine the number of rows and columns
	nbRows := len(pageData)
	nbCols := 0
	if nbRows > 0 {
		nbCols = len(pageData[0])
	}

	// Resize the table and set the data
	p.TableBinding.Resize(nbRows, nbCols)
	if nbRows > 0 {
		_ = p.TableBinding.Set(pageData)
	}
}

// setState sets the current state of the paginator.
func (p *Paginator) setState(newState paginatorState) {
	p.currentState = newState
}

// paginatorState defines the interface for paginator states.
// Each state implementation encapsulates the pagination behavior for a specific mode
type paginatorState interface {
	// Next advances to the next page. Returns true if more pages are available.
	Next() bool

	// Prev moves to the previous page. Returns true if the operation was successful.
	Prev() bool

	// PageNumber returns the current page number (1-based).
	PageNumber() int

	// TotalPages returns the total number of pages, or -1 if unknown.
	TotalPages() int

	// CurrentPageSize returns the number of rows in the current page.
	CurrentPageSize() int

	// HasNext returns true if there are more pages available.
	HasNext() bool

	// CurrentIndex returns the index of the first data row to display.
	CurrentIndex() int

	// PageSize returns the effective page size for this state.
	PageSize() int

	// Records returns all records currently loaded in this state.
	// For lazy loading, this includes only loaded pages.
	// For non-lazy loading, this includes all file records.
	Records() []Record
}

// paginatorInitialized is the initial state before data is loaded.
// All operations return default values as no data is available.
type paginatorInitialized struct{}

func (*paginatorInitialized) Next() bool           { return false }
func (*paginatorInitialized) Prev() bool           { return false }
func (*paginatorInitialized) PageNumber() int      { return 0 }
func (*paginatorInitialized) TotalPages() int      { return 0 }
func (*paginatorInitialized) CurrentPageSize() int { return 0 }
func (*paginatorInitialized) HasNext() bool        { return false }
func (*paginatorInitialized) CurrentIndex() int    { return 0 }
func (*paginatorInitialized) PageSize() int        { return defaultPageSize }
func (*paginatorInitialized) Records() []Record    { return []Record{} }

// paginatorEagerState is the state for non-lazy loading (all data loaded upfront).
// This state reads all CSV data on initialization and paginates through it.
type paginatorEagerState struct {
	p *Paginator

	// content is the CSV reader (exhausted after reading all data)
	content *csv.Reader

	// rawStartIndex is the index of the first record in the current view
	// (data rows only, excluding header)
	rawStartIndex int

	// records holds all records from the CSV file
	records []Record
}

// newEagerPaginatorState creates a new eager loading state and reads all data from CSV.
func newEagerPaginatorState(p *Paginator, r *csv.Reader) *paginatorEagerState {
	s := &paginatorEagerState{
		p:       p,
		content: r,
	}

	// Read all records from the CSV
	for {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			// For other errors, just break
			break
		}
		s.records = append(s.records, Record(record))
	}

	// Initialize state
	s.rawStartIndex = 0

	// Update table binding with first page
	s.p.updateTableBinding(s.records)

	return s
}

func (s *paginatorEagerState) Next() bool {
	if !s.HasNext() {
		return false
	}
	s.rawStartIndex += s.PageSize()
	s.p.updateTableBinding(s.records)
	return s.HasNext()
}

func (s *paginatorEagerState) Prev() bool {
	if s.rawStartIndex <= 0 {
		return false
	}
	s.rawStartIndex -= s.PageSize()
	if s.rawStartIndex < 0 {
		s.rawStartIndex = 0
	}
	s.p.updateTableBinding(s.records)
	return true
}

func (s *paginatorEagerState) PageNumber() int {
	if s.PageSize() == 0 {
		return 0
	}
	return (s.rawStartIndex / s.p.PageSize) + 1
}

func (s *paginatorEagerState) TotalPages() int {
	if s.PageSize() == 0 {
		return 0
	}
	// Total pages based on all records
	return (len(s.records) + s.PageSize() - 1) / s.PageSize()
}

func (s *paginatorEagerState) CurrentPageSize() int {
	pageSize := s.PageSize()
	startIndex := s.CurrentIndex()

	if startIndex >= len(s.records) {
		return 0
	}

	remaining := len(s.records) - startIndex
	if remaining < pageSize {
		return remaining
	}
	return pageSize
}

func (s *paginatorEagerState) HasNext() bool {
	pageSize := s.PageSize()
	startIndex := s.CurrentIndex()
	return startIndex+pageSize < len(s.records)
}

func (s *paginatorEagerState) CurrentIndex() int {
	// For eager mode, current index is the starting data row index
	return s.rawStartIndex
}

func (s *paginatorEagerState) PageSize() int {
	// For eager mode, page size is the configured page size
	return s.p.PageSize
}

func (s *paginatorEagerState) Records() []Record {
	return s.records
}

// paginatorLazyState is the state for lazy loading (loads data on demand).
// This state cannot go back to previous pages because csv.Reader doesn't support seeking.
// Only loads data as needed, keeping memory usage low for large files.
type paginatorLazyState struct {
	paginatorEagerState
}

// newLazyPaginatorState creates a new lazy loading state.
// It loads the first page of data immediately.
func newLazyPaginatorState(p *Paginator, r *csv.Reader) *paginatorLazyState {
	// Create base eager state structure
	base := &paginatorEagerState{
		p:             p,
		content:       r,
		rawStartIndex: 0,
		records:       []Record{},
	}

	// Load first page
	for i := 0; i < p.PageSize; i++ {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
		base.records = append(base.records, Record(record))
	}

	// Create lazy state with the loaded data
	state := &paginatorLazyState{
		paginatorEagerState: *base,
	}

	// Update table binding with first page
	state.paginatorEagerState.rawStartIndex = 0
	state.p.updateTableBinding(base.records)

	return state
}

func (s *paginatorLazyState) TotalPages() int {
	// For lazy loading, we can't determine the total number of pages
	return -1
}

func (s *paginatorLazyState) Next() bool {
	// For lazy loading, always try to load the next page
	if !s.loadNextPage() {
		// Couldn't load more data, we're at the end
		return false
	}

	s.rawStartIndex += s.p.PageSize
	s.p.updateTableBinding(s.records)
	return true
}

func (s *paginatorLazyState) Prev() bool {
	// For lazy loading with csv.Reader, we can't go back because csv.Reader doesn't support seeking
	// In a real implementation, we would need to store the underlying seekable content
	// and re-read from the beginning. For now, we disable previous page navigation.
	return false
}

func (s *paginatorLazyState) CurrentPageSize() int {
	// For lazy loading, return the number of data rows currently loaded
	// that should be displayed on the current page
	pageSize := s.PageSize()
	startIndex := s.CurrentIndex()

	if startIndex >= len(s.records) {
		return 0
	}

	remaining := len(s.records) - startIndex
	if remaining < pageSize {
		return remaining
	}
	return pageSize
}

func (s *paginatorLazyState) HasNext() bool {
	// For lazy loading, we can always try to load the next page
	// The actual check happens in Next() when we try to load
	return true
}

// loadNextPage loads the next page of data from the CSV reader.
// Returns true if more data was loaded, false if we reached the end.
func (s *paginatorLazyState) loadNextPage() bool {
	for i := 0; i < s.p.PageSize; i++ {
		record, err := s.p.content.Read()
		if err != nil {
			return false
		}
		s.records = append(s.records, Record(record))
	}
	return true
}

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

	// PageNumber is the current page number (1-based).
	PageNumber int

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
	if tableBinding == nil {
		panic("tableBinding cannot be nil")
	}
	hasHeader := binding.NewBool()

	p := &Paginator{
		PageSize:     defaultPageSize,
		PageNumber:   1,
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
		state := newLazyPaginatorState(p, r)
		p.setState(state)
	} else {
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

// TotalPages returns the total number of pages.
// For lazy loading, returns -1 to indicate unknown total (displayed as "?").
func (p *Paginator) TotalPages() int {
	return p.currentState.TotalPages()
}

// HasNext returns true if there are more pages available.
func (p *Paginator) HasNext() bool {
	return p.currentState.HasNext()
}

// BaseIndex returns the theoretical index display of the first data row displayed on the page when no header row is present.
func (p *Paginator) BaseIndex() int {
	return p.PageSize * (p.PageNumber - 1)
}

// DataIndex returns the index of the first data row displayed on the current page under the header row (if present).
func (p *Paginator) DataIndex() int {
	if !p.hasHeader() {
		return p.BaseIndex()
	}

	switch p.PageNumber {
	case 1:
		return 1
	case 2:
		return p.BaseIndex()
	default:
		return p.BaseIndex() - p.PageNumber + 2
	}
}

// hasHeader returns whether the paginator should display a header row.
func (p *Paginator) hasHeader() bool {
	return u.SkipV(p.HasHeader.Get())
}

// updateTableBinding updates the table binding with the current page of data.
// It considers the header preference and current pagination state.
func (p *Paginator) updateTableBinding(allRecords []Record) {
	if len(allRecords) == 0 {
		p.TableBinding.Resize(0, 0)
		return
	}

	// Get pagination parameters
	nbDataRows := p.pageDataSize(allRecords)
	dataIndex := p.DataIndex()
	hasHeader := p.hasHeader()

	// Determine the data rows to display
	var pageData [][]string

	// Calculate effective start and end indices
	end := dataIndex + nbDataRows
	if end > len(allRecords) {
		end = len(allRecords)
	}

	// Include header row if enabled
	if hasHeader && len(allRecords) > 0 {
		pageData = append(pageData, []string(allRecords[0]))
		if dataIndex < len(allRecords) {
			for i := range dataIndex - 1 {
				pageData = append(pageData, []string(allRecords[i]))
			}
		}
	} else {
		if dataIndex < len(allRecords) {
			for i := dataIndex; i < end; i++ {
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
		u.Skip(p.TableBinding.Set(pageData))
	}
}

// setState sets the current state of the paginator.
func (p *Paginator) setState(newState paginatorState) {
	p.currentState = newState
}

func (p *Paginator) pageDataSize(records []Record) int {
	pageSize := p.PageSize
	dataIndex := p.DataIndex()

	if dataIndex >= len(records) {
		return 0
	}

	remaining := len(records) - dataIndex
	if remaining < pageSize {
		return remaining
	}
	return pageSize
}

// paginatorState defines the interface for paginator states.
// Each state implementation encapsulates the pagination behavior for a specific mode
type paginatorState interface {
	// Next advances to the next page. Returns true if more pages are available.
	Next() bool

	// Prev moves to the previous page. Returns true if the operation was successful.
	Prev() bool

	// TotalPages returns the total number of pages, or -1 if unknown.
	TotalPages() int

	// HasNext returns true if there are more pages available.
	HasNext() bool

	// Records returns all records currently loaded in this state.
	// For lazy loading, this includes only loaded pages.
	// For non-lazy loading, this includes all file records.
	Records() []Record
}

// paginatorInitialized is the initial state before data is loaded.
// All operations return default values as no data is available.
type paginatorInitialized struct{}

func (*paginatorInitialized) Next() bool        { return false }
func (*paginatorInitialized) Prev() bool        { return false }
func (*paginatorInitialized) TotalPages() int   { return 0 }
func (*paginatorInitialized) HasNext() bool     { return false }
func (*paginatorInitialized) Records() []Record { return []Record{} }

// paginatorEagerState is the state for non-lazy loading (all data loaded upfront).
// This state reads all CSV data on initialization and paginates through it.
type paginatorEagerState struct {
	p *Paginator

	// content is the CSV reader (exhausted after reading all data)
	content *csv.Reader

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

	// Update table binding with first page
	s.p.updateTableBinding(s.records)

	return s
}

func (s *paginatorEagerState) Next() bool {
	if !s.HasNext() {
		return false
	}
	s.p.PageNumber++
	s.p.updateTableBinding(s.records)
	return s.HasNext()
}

func (s *paginatorEagerState) Prev() bool {
	if s.p.PageNumber == 1 {
		return false
	}
	s.p.PageNumber--
	s.p.updateTableBinding(s.records)
	return s.p.PageNumber > 1
}

func (s *paginatorEagerState) TotalPages() int {
	if s.p.PageSize == 0 {
		return 0
	}
	return (len(s.records) + s.p.PageSize - 1) / s.p.PageSize
}

func (s *paginatorEagerState) HasNext() bool {
	pageSize := s.p.PageSize
	startIndex := s.p.DataIndex()
	return startIndex+pageSize < len(s.records)
}

func (s *paginatorEagerState) Records() []Record {
	return s.records
}

// paginatorLazyState is the state for lazy loading (loads data on demand).
// This state cannot go back to previous pages because csv.Reader doesn't support seeking.
// Only loads data as needed, keeping memory usage low for large files.
type paginatorLazyState struct {
	paginatorEagerState
	endReached bool
}

// newLazyPaginatorState creates a new lazy loading state.
// It loads the first page of data immediately.
func newLazyPaginatorState(p *Paginator, r *csv.Reader) *paginatorLazyState {
	base := &paginatorEagerState{
		p:       p,
		content: r,
		records: []Record{},
	}

	// Load first page
	for range p.PageSize {
		record, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
		base.records = append(base.records, Record(record))
	}

	state := &paginatorLazyState{
		paginatorEagerState: *base,
	}

	state.p.updateTableBinding(base.records)

	return state
}

func (s *paginatorLazyState) TotalPages() int {
	if s.endReached {
		return s.paginatorEagerState.TotalPages()
	}
	return -1
}

func (s *paginatorLazyState) Next() bool {
	prevNbRecs := len(s.records)
	hasMore := s.loadNextPage()
	if len(s.records) == prevNbRecs {
		// No new records were loaded, so we reached the end
		return false
	}

	s.p.PageNumber++
	s.p.updateTableBinding(s.records)
	return hasMore
}

func (s *paginatorLazyState) HasNext() bool {
	if s.endReached {
		totalPageNb := s.paginatorEagerState.TotalPages()
		return s.p.PageNumber < totalPageNb
	}
	return true
}

// loadNextPage loads the next page of data from the CSV reader.
// Returns true if more data was loaded, false if we reached the end.
func (s *paginatorLazyState) loadNextPage() bool {
	for range s.p.PageSize {
		record, err := s.p.content.Read()
		if err != nil {
			s.endReached = true
			return false
		}
		s.records = append(s.records, Record(record))
	}
	return true
}

package csveditor

import (
	"encoding/csv"

	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/u"
)

const (
	defaultPageSize = 10
)

type Record []string

type Paginator struct {
	// Records   []Record
	PageSize  int
	HasHeader binding.Bool
	binding   binding.List[[]binding.String]
	// // rawStartIndex is the index of the first record in a default view (without header enabled)
	// rawStartIndex int

	currentState paginatorState
}

func NewCsvPaginator(bound binding.List[[]binding.String]) *Paginator {
	hasHeader := binding.NewBool()

	p := &Paginator{
		PageSize:  defaultPageSize,
		binding:   bound,
		HasHeader: hasHeader,
	}
	p.setState(&paginatorInitialized{})

	hasHeader.AddListener(binding.NewDataListener(p.updateBinding))

	return p
}

func (p *Paginator) MarksReady(r *csv.Reader, readOnly bool) {
	p.Records = []Record{}
	// p.rawStartIndex = 0

	stateReady := &paginatorReady{
		content: r,
		p:       p,
	}
	if readOnly {
		p.setState(&paginatorLazyReady{
			paginatorReady: *stateReady,
		})
	} else {
		p.setState(stateReady)

	}
}

// Reset is	deprecated
// func (p *Paginator) Reset() {
// 	p.Records = []Record{}
// 	p.rawStartIndex = 0
// }

// Append is deprecated
func (p *Paginator) Append(vals []string) {
	// p.Records = append(p.Records, Record(vals))
	// p.updateBinding()
}

func (p *Paginator) Next() bool {
	return p.currentState.Next()
}

func (p *Paginator) Prev() bool {
	return p.currentState.Prev()
}

func (p *Paginator) PageNumber() int {
	return p.currentState.PageNumber()
}

func (p *Paginator) TotalPages() int {
	return p.currentState.TotalPages()
}

func (p *Paginator) CurrentPageSize() int {
	if !p.HasNext() {
		return len(p.Records) - p.rawStartIndex
	}
	return p.PageSize
}

func (p *Paginator) HasNext() bool {
	return p.CurrentIndex()+p.pageSize() < len(p.Records)
}

func (p *Paginator) pageSize() int {
	pageSize := p.PageSize
	if p.rawStartIndex > 0 && p.hasHeader() {
		pageSize--
	}
	return pageSize
}

// CurrentIndex returns the actual index of the first record to display.
// Header row is excluded (except of the first page).
func (p *Paginator) CurrentIndex() int {
	index := p.rawStartIndex
	pageIndex := p.pageIndex()
	if index > 0 && p.hasHeader() {
		if pageIndex > 1 {
			index -= pageIndex - 1
		}
	}
	return index
}

func (p *Paginator) hasHeader() bool {
	return u.SkipV(p.HasHeader.Get())
}

func (p *Paginator) updateBinding(rowsToDisplay []Record) {
	if p.binding == nil {
		return
	}

	pageSize := p.pageSize()
	startIndex := p.CurrentIndex()
	hasHeader := p.hasHeader()

	if len(rowsToDisplay) >= p.binding.Length() {
		for i, records := range rowsToDisplay {
			row := u.SkipV(p.binding.GetValue(i))
			if len(records) > len(row) {
				for range len(records) - len(row) {
					row = append(row, binding.NewString())
				}
				u.Skip(p.binding.SetValue(i, row))
			} else if len(records) < len(row) {
				row = row[:len(records)]
				u.Skip(p.binding.SetValue(i, row))
			}
			for j, cellItem := range row {
				u.Skip(cellItem.Set(rowsToDisplay[i][j]))
			}
		}
		for range len(rowsToDisplay) - p.binding.Length() {
			p.binding.Append(make([]binding.String, ))
		}
	} else if len(rowsToDisplay) > p.binding.Length() {
		for i, _ := range rowsToDisplay {
			row := u.SkipV(p.binding.GetValue(i))
			if len(rowsToDisplay[i]) > len(row) {
				for range len(rowsToDisplay[i]) - len(row) {
					row = append(row, binding.NewString())
				}
				u.Skip(p.binding.SetValue(i, row))
			} else if len(rowsToDisplay[i]) < len(row) {
				row = row[:len(rowsToDisplay[i])]
				u.Skip(p.binding.SetValue(i, row))
			}
			for j, cellItem := range row {
				u.Skip(cellItem.Set(rowsToDisplay[i][j]))
			}
		}
	} else { // len(rowsToDisplay) < p.binding.Length()

	}

	end := startIndex + pageSize
	if end > len(p.Records) {
		end = len(p.Records)
	}

	if startIndex >= len(p.Records) {
		u.Skip(p.binding.Set(nil))
		return
	}

	var page [][]string
	if hasHeader && p.rawStartIndex > 0 {
		page = append(page, []string(p.Records[0]))
	}
	for i := startIndex; i < end; i++ {
		page = append(page, []string(p.Records[i]))
	}
	u.Skip(p.binding.Set(page))
}

// func (p *Paginator) pageIndex() int {
// 	if p.PageSize == 0 {
// 		return 0
// 	}
// 	return p.rawStartIndex / p.PageSize
// }

func (p *Paginator) setState(newState paginatorState) {
	p.currentState = newState
}

type paginatorState interface {
	Next() bool
	Prev() bool
	PageNumber() int
	TotalPages() int
	CurrentPageSize() int
	HasNext() bool
}

type paginatorInitialized struct{}

func (*paginatorInitialized) Next() bool { return false }

func (*paginatorInitialized) Prev() bool { return false }

func (*paginatorInitialized) PageNumber() int { return 0 }

func (*paginatorInitialized) TotalPages() int { return 0 }

func (*paginatorInitialized) HasNext() bool { return false }

func (*paginatorInitialized) CurrentPageSize() int { return 0 }

type paginatorReady struct {
	p       *Paginator
	content *csv.Reader
	// rawStartIndex is the index of the first record in a default view (without header enabled)
	rawStartIndex int
	records       []Record
}

func newPaginatorStateReady(p *Paginator, r *csv.Reader) paginatorState {
	s := &paginatorReady{
		p:       p,
		content: r,
	}

	nbRows := 0
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		nbRows++
		s.records = append(s.records, Record(record))
	}
	p.updateBinding(s.records)

	// if len(s.records) == 0 {
	// 	return
	// }

	return s
}

func (s *paginatorReady) Next() bool {
	if !s.p.HasNext() {
		return false
	}
	s.rawStartIndex += s.p.PageSize
	s.p.updateBinding()
	return s.p.HasNext()
}

func (s *paginatorReady) Prev() bool {
	if s.rawStartIndex <= 0 {
		return false
	}
	s.rawStartIndex -= s.p.PageSize
	if s.rawStartIndex < 0 {
		s.rawStartIndex = 0
	}
	s.p.updateBinding()
	return true
}

func (s *paginatorReady) PageNumber() int {
	return s.pageIndex() + 1
}

func (s *paginatorReady) TotalPages() int {
	if s.p.PageSize == 0 {
		return 0
	}
	total := len(s.p.Records) / s.p.PageSize
	if len(s.p.Records)%s.p.PageSize != 0 {
		total++
	}
	return total
}

func (s *paginatorReady) HasNext() bool {
	return false
}

func (s *paginatorReady) CurrentPageSize() int {
	return 0
}

func (s *paginatorReady) pageIndex() int {
	if s.p.PageSize == 0 {
		return 0
	}
	return s.rawStartIndex / s.p.PageSize
}

type paginatorLazyReady struct {
	paginatorReady
}

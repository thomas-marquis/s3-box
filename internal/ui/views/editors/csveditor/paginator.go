package csveditor

import (
	"fyne.io/fyne/v2/data/binding"
	"github.com/thomas-marquis/s3-box/internal/u"
)

const (
	defaultPageSize = 10
)

type Record []string

type Paginator struct {
	Records   []Record
	PageSize  int
	HasHeader binding.Bool
	binding   binding.List[[]string]
	// rawStartIndex is the index of the first record in a default view (without header enabled)
	rawStartIndex int
}

func NewCsvPaginator(bound binding.List[[]string]) *Paginator {
	hasHeader := binding.NewBool()

	p := &Paginator{
		PageSize:  defaultPageSize,
		binding:   bound,
		HasHeader: hasHeader,
	}

	hasHeader.AddListener(binding.NewDataListener(p.updateBinding))

	return p
}

func (p *Paginator) Reset() {
	p.Records = []Record{}
	p.rawStartIndex = 0
}

func (p *Paginator) Append(vals []string) {
	p.Records = append(p.Records, Record(vals))
	p.updateBinding()
}

func (p *Paginator) Next() bool {
	if !p.HasNext() {
		return false
	}
	p.rawStartIndex += p.PageSize
	p.updateBinding()
	return p.HasNext()
}

func (p *Paginator) Prev() bool {
	if p.rawStartIndex <= 0 {
		return false
	}
	p.rawStartIndex -= p.PageSize
	if p.rawStartIndex < 0 {
		p.rawStartIndex = 0
	}
	p.updateBinding()
	return true
}

func (p *Paginator) PageNumber() int {
	return p.pageIndex() + 1
}

func (p *Paginator) TotalPages() int {
	if p.PageSize == 0 {
		return 0
	}
	total := len(p.Records) / p.PageSize
	if len(p.Records)%p.PageSize != 0 {
		total++
	}
	return total
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

func (p *Paginator) updateBinding() {
	if p.binding == nil {
		return
	}

	pageSize := p.pageSize()
	startIndex := p.CurrentIndex()
	hasHeader := p.hasHeader()

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

func (p *Paginator) pageIndex() int {
	if p.PageSize == 0 {
		return 0
	}
	return p.rawStartIndex / p.PageSize
}

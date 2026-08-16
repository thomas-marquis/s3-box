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
	Records      []Record
	CurrentIndex int
	PageSize     int
	binding      binding.List[[]string]
	HasHeader    binding.Bool
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

func (p *Paginator) Append(vals []string) {
	p.Records = append(p.Records, Record(vals))
	p.updateBinding()
}

func (p *Paginator) Next() bool {
	if p.currentIndex()+p.pageSize() >= len(p.Records) {
		return false
	}
	p.CurrentIndex += p.PageSize
	p.updateBinding()
	return p.currentIndex()+p.pageSize() < len(p.Records)
}

func (p *Paginator) Prev() bool {
	if p.CurrentIndex <= 0 {
		return false
	}
	p.CurrentIndex -= p.PageSize
	if p.CurrentIndex < 0 {
		p.CurrentIndex = 0
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
	if p.CurrentIndex+p.PageSize >= len(p.Records) {
		return len(p.Records) - p.CurrentIndex
	}
	return p.PageSize
}

func (p *Paginator) HasNext() bool {
	return p.CurrentIndex+p.PageSize < len(p.Records)
}

func (p *Paginator) pageSize() int {
	pageSize := p.PageSize
	if p.hasHeader() {
		pageSize--
	}
	return pageSize
}

func (p *Paginator) currentIndex() int {
	index := p.CurrentIndex
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
	startIndex := p.currentIndex()
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
	if hasHeader {
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
	return p.CurrentIndex / p.PageSize
}

package csveditor

import (
	"fyne.io/fyne/v2/data/binding"
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
}

func NewCsvPaginator(bound binding.List[[]string]) *Paginator {
	return &Paginator{
		PageSize: defaultPageSize,
		binding:  bound,
	}
}

func (p *Paginator) Append(vals []string) {
	p.Records = append(p.Records, Record(vals))
	p.updateBinding()
}

func (p *Paginator) Next() bool {
	if p.CurrentIndex+p.PageSize >= len(p.Records) {
		return false
	}
	p.CurrentIndex += p.PageSize
	p.updateBinding()
	return p.CurrentIndex+p.PageSize < len(p.Records)
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
	if p.PageSize == 0 {
		return 0
	}
	return p.CurrentIndex/p.PageSize + 1
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

func (p *Paginator) updateBinding() {
	if p.binding == nil {
		return
	}

	end := p.CurrentIndex + p.PageSize
	if end > len(p.Records) {
		end = len(p.Records)
	}

	if p.CurrentIndex >= len(p.Records) {
		_ = p.binding.Set(nil)
		return
	}

	var page [][]string
	for i := p.CurrentIndex; i < end; i++ {
		page = append(page, []string(p.Records[i]))
	}
	_ = p.binding.Set(page)
}

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
	bound        binding.List[[]string]
}

func NewCsvPaginator(bound binding.List[[]string]) *Paginator {
	return &Paginator{
		PageSize: defaultPageSize,
		bound:    bound,
	}
}

func (p *Paginator) Append(vals []string) {
	p.Records = append(p.Records, Record(vals))
	p.updateBound()
}

func (p *Paginator) Next() bool {
	if p.CurrentIndex+p.PageSize >= len(p.Records) {
		return false
	}
	p.CurrentIndex += p.PageSize
	p.updateBound()
	return true
}

func (p *Paginator) Prev() bool {
	if p.CurrentIndex <= 0 {
		return false
	}
	p.CurrentIndex -= p.PageSize
	if p.CurrentIndex < 0 {
		p.CurrentIndex = 0
	}
	p.updateBound()
	return true
}

func (p *Paginator) updateBound() {
	if p.bound == nil {
		return
	}

	end := p.CurrentIndex + p.PageSize
	if end > len(p.Records) {
		end = len(p.Records)
	}

	if p.CurrentIndex >= len(p.Records) {
		_ = p.bound.Set(nil)
		return
	}

	var page [][]string
	for i := p.CurrentIndex; i < end; i++ {
		page = append(page, []string(p.Records[i]))
	}
	_ = p.bound.Set(page)
}

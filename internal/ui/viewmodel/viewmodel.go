package viewmodel

import "fyne.io/fyne/v2/data/binding"

type ViewModel interface {
	Loading() binding.Bool

	IsLoading() bool

	ErrorMessage() binding.String

	InfoMessage() binding.String
}

type baseViewModel struct {
	loading      binding.Bool
	errorMessage binding.String
	infoMessage  binding.String
}

func (b *baseViewModel) Loading() binding.Bool {
	return b.loading
}

func (b *baseViewModel) IsLoading() bool {
	val, _ := b.loading.Get()
	return val
}

func (b *baseViewModel) ErrorMessage() binding.String {
	return b.errorMessage
}

func (b *baseViewModel) InfoMessage() binding.String {
	return b.infoMessage
}

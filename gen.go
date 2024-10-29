package main

import _ "go.uber.org/mock/gomock"

//go:generate mockgen -package mocks_explorer -destination mocks/explorer/repository.go go2s3/internal/explorer Repository
//go:generate mockgen -package mocks_appcontext -destination mocks/context/appcontext.go go2s3/internal/ui/app/context AppContext

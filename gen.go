package main

import _ "go.uber.org/mock/gomock"

//go:generate mockgen -package mocks_explorer -destination mocks/explorer/directory_repository.go github.com/thomas-marquis/s3-box/internal/explorer S3DirectoryRepository
//go:generate mockgen -package mocks_appcontext -destination mocks/context/appcontext.go github.com/thomas-marquis/s3-box/internal/ui/app/context AppContext

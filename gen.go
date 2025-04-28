package main

import _ "go.uber.org/mock/gomock"

//go:generate mockgen -package mocks_explorer -destination mocks/explorer/directory_repository.go github.com/thomas-marquis/s3-box/internal/explorer S3DirectoryRepository
//go:generate mockgen -package mocks_explorer -destination mocks/explorer/file_s3_repository.go github.com/thomas-marquis/s3-box/internal/explorer S3FileRepository
//go:generate mockgen -package mocks_appcontext -destination mocks/context/appcontext.go github.com/thomas-marquis/s3-box/internal/ui/app/context AppContext
//go:generate mockgen -package mocks_connection -destination mocks/connection/connection_repository.go github.com/thomas-marquis/s3-box/internal/connection Repository
//go:generate mockgen -package mocks_settings -destination mocks/settings/settings_repository.go github.com/thomas-marquis/s3-box/internal/settings Repository
//go:generate mockgen -package mocks_binding -destination mocks/binding/tree.go fyne.io/fyne/v2/data/binding UntypedTree

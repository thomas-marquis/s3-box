package main

import _ "go.uber.org/mock/gomock"

// Domain
//go:generate mockgen -package mocks_explorer -destination mocks/directory/repository.go github.com/thomas-marquis/s3-box/internal/domain/directory Repository
//go:generate mockgen -package mocks_connection_deck -destination mocks/connection_deck/repository.go github.com/thomas-marquis/s3-box/internal/domain/connection_deck Repository
//go:generate mockgen -package mocks_settings -destination mocks/settings/repository.go github.com/thomas-marquis/s3-box/internal/domain/settings Repository
//go:generate mockgen -package mocks_notification -destination mocks/notification/repository.go github.com/thomas-marquis/s3-box/internal/domain/notification Repository

// View Model
//go:generate mockgen -package mocks_viewmodel -destination mocks/viewmodel/explorer_viewmodel.go github.com/thomas-marquis/s3-box/internal/ui/viewmodel ExplorerViewModel
//go:generate mockgen -package mocks_viewmodel -destination mocks/viewmodel/connection_viewmodel.go github.com/thomas-marquis/s3-box/internal/ui/viewmodel ConnectionViewModel
//go:generate mockgen -package mocks_viewmodel -destination mocks/viewmodel/settings_viewmodel.go github.com/thomas-marquis/s3-box/internal/ui/viewmodel SettingsViewModel
//go:generate mockgen -package mocks_viewmodel -destination mocks/viewmodel/notification_viewmodel.go github.com/thomas-marquis/s3-box/internal/ui/viewmodel NotificationViewModel

// Global
//go:generate mockgen -package mocks_appcontext -destination mocks/context/appcontext.go github.com/thomas-marquis/s3-box/internal/ui/app/context AppContext
//go:generate mockgen -package mocks_event -destination mocks/event/bus.go github.com/thomas-marquis/s3-box/internal/domain/shared/event Bus

// External
//go:generate mockgen -package mocks_binding -destination mocks/binding/tree.go fyne.io/fyne/v2/data/binding UntypedTree

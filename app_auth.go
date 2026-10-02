// app_auth.go - account management Wails bindings.

package main

import (
	"github.com/Aether-Launcher/Aether/pkg/auth"
)

func (a *App) GetActiveAccount() *auth.Account {
	return auth.GetActiveAccount()
}

// GetAccounts returns all saved accounts
func (a *App) GetAccounts() []auth.Account {
	return auth.GetAccounts()
}

// LoginOffline creates or switches to an offline account with the given username
func (a *App) LoginOffline(username string) (auth.Account, error) {
	return auth.AddOfflineAccount(username)
}

// StartMicrosoftAuth initiates the Microsoft OAuth2 PKCE login flow
func (a *App) StartMicrosoftAuth() (auth.Account, error) {
	acc, err := auth.StartPKCEAuthFlow(a.ctx)
	if err != nil {
		return auth.Account{}, err
	}
	if err := auth.AddMicrosoftAccount(*acc); err != nil {
		return auth.Account{}, err
	}
	return *acc, nil
}

// SetActiveAccount sets the active account by ID
func (a *App) SetActiveAccount(id string) error {
	return auth.SetActiveAccount(id)
}

// RemoveAccount removes an account by ID
func (a *App) RemoveAccount(id string) error {
	return auth.RemoveAccount(id)
}

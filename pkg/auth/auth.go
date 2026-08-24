package auth

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"Aether/pkg/fs"
	"github.com/zalando/go-keyring"
)

// AccountType defines the type of account
type AccountType string

const (
	TypeOffline   AccountType = "offline"
	TypeMicrosoft AccountType = "microsoft"
)

// Account represents a user profile
type Account struct {
	ID           string      `json:"id"`
	Type         AccountType `json:"type"`
	Username     string      `json:"username"`
	AccessToken  string      `json:"accessToken,omitempty"`
	RefreshToken string      `json:"refreshToken,omitempty"`
	ExpiresAt    int64       `json:"expiresAt,omitempty"`
}

// AccountStore represents the accounts.json file structure
type AccountStore struct {
	ActiveAccountID string    `json:"activeAccountId"`
	Accounts        []Account `json:"accounts"`
}

var store AccountStore
var storeMu sync.Mutex

func getStorePath() string {
	return filepath.Join(fs.GetDataDir(), "accounts.json")
}

// LoadAccounts reads accounts.json into memory
func LoadAccounts() error {
	path := getStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			store = AccountStore{Accounts: []Account{}}
			return nil
		}
		return err
	}
	if err := json.Unmarshal(data, &store); err != nil {
		// Backup corrupt file and reset
		_ = os.WriteFile(path+".corrupt", data, 0600)
		store = AccountStore{Accounts: []Account{}}
		return nil
	}
	return nil
}

// SaveAccounts writes the in-memory store to accounts.json atomically
func SaveAccounts() error {
	path := getStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// GenerateOfflineUUID generates a deterministic UUID for offline mode based on username
func GenerateOfflineUUID(username string) string {
	hash := md5.Sum([]byte("OfflinePlayer:" + username))
	hash[6] = (hash[6] & 0x0f) | 0x30
	hash[8] = (hash[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// AddOfflineAccount creates a new offline account and sets it as active
func AddOfflineAccount(username string) (Account, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	if err := LoadAccounts(); err != nil {
		return Account{}, err
	}

	id := GenerateOfflineUUID(username)

	for _, acc := range store.Accounts {
		if acc.ID == id {
			store.ActiveAccountID = id
			_ = SaveAccounts()
			return acc, nil
		}
	}

	newAcc := Account{
		ID:       id,
		Type:     TypeOffline,
		Username: username,
	}

	store.Accounts = append(store.Accounts, newAcc)
	store.ActiveAccountID = id
	err := SaveAccounts()
	return newAcc, err
}

// AddMicrosoftAccount saves a Microsoft account and sets it as active
func AddMicrosoftAccount(acc Account) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	if err := LoadAccounts(); err != nil {
		return err
	}

	// Replace if exists
	for i, existing := range store.Accounts {
		if existing.ID == acc.ID {
			store.Accounts[i] = acc
			store.ActiveAccountID = acc.ID
			return SaveAccounts()
		}
	}

	store.Accounts = append(store.Accounts, acc)
	store.ActiveAccountID = acc.ID
	return SaveAccounts()
}


// RemoveAccount removes an account by ID
func RemoveAccount(id string) error {
	if err := LoadAccounts(); err != nil {
		return err
	}

	filtered := store.Accounts[:0]
	for _, acc := range store.Accounts {
		if acc.ID != id {
			filtered = append(filtered, acc)
		}
	}
	store.Accounts = filtered

	if store.ActiveAccountID == id {
		if len(store.Accounts) > 0 {
			store.ActiveAccountID = store.Accounts[0].ID
		} else {
			store.ActiveAccountID = ""
		}
	}

	_ = keyring.Delete(KeyringService, id)
	return SaveAccounts()
}

// GetActiveAccount returns the currently active account, or nil if none
func GetActiveAccount() *Account {
	if err := LoadAccounts(); err != nil {
		return nil
	}
	for i, acc := range store.Accounts {
		if acc.ID == store.ActiveAccountID {
			return &store.Accounts[i]
		}
	}
	return nil
}

// GetAccounts returns all saved accounts
func GetAccounts() []Account {
	if err := LoadAccounts(); err != nil {
		return []Account{}
	}
	return store.Accounts
}

// SetActiveAccount sets the active account by ID
func SetActiveAccount(id string) error {
	if err := LoadAccounts(); err != nil {
		return err
	}
	for _, acc := range store.Accounts {
		if acc.ID == id {
			store.ActiveAccountID = id
			return SaveAccounts()
		}
	}
	return fmt.Errorf("account not found")
}

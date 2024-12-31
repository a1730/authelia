package authentication

import (
	"context"

	"github.com/go-crypt/crypt/algorithm"
	// "github.com/go-crypt/crypt/algorithm/argon2"
	// "github.com/go-crypt/crypt/algorithm/bcrypt"
	// "github.com/go-crypt/crypt/algorithm/pbkdf2"
	// "github.com/go-crypt/crypt/algorithm/scrypt"
	// "github.com/go-crypt/crypt/algorithm/shacrypt".

	"github.com/authelia/authelia/v4/internal/configuration/schema"
	"github.com/authelia/authelia/v4/internal/model"
	"github.com/authelia/authelia/v4/internal/storage"
	// "github.com/authelia/authelia/v4/internal/logging".
)

// DBUserProvider is a provider reading details from a sql database.
type DBUserProvider struct {
	config   *schema.AuthenticationBackendSQL
	database storage.Provider
	hash     algorithm.Hash
}

// NewDBUserProvider creates a new instance of FileUserProvider.
func NewDBUserProvider(config *schema.AuthenticationBackendSQL, database storage.Provider) (provider *DBUserProvider) {
	return &DBUserProvider{
		config:   config,
		database: database,
	}
}

// StartupCheck implements authentication.UserProvider.StartupCheck().
func (p *DBUserProvider) StartupCheck() (err error) {
	if err := p.database.StartupCheck(); err != nil {
		return err
	}

	// TODO: verify that table exists.

	if p.hash, err = NewFileCryptoHashFromConfig(p.config.Password); err != nil {
		return err
	}

	return nil
}

// CheckUserPassword implements authentication.UserProvider.CheckUserPassword().
func (p *DBUserProvider) CheckUserPassword(username string, password string) (valid bool, err error) {
	var user *model.User

	if user, err = p.loadUser(username); err != nil {
		return false, err
	}

	return user.Password.MatchAdvanced(password)
}

// GetDetails implements authentication.UserProvider.GetDetails().
func (p *DBUserProvider) GetDetails(username string) (details *UserDetails, err error) {
	var user *model.User

	if user, err = p.loadUser(username); err != nil {
		return nil, err
	}

	// TODO: refactor to a mapper function.
	return &UserDetails{
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Emails:      []string{user.Email},
		Groups:      user.Groups,
	}, nil
}

// UpdatePassword implements authentication.UserProvider.UpdatePassword().
func (p *DBUserProvider) UpdatePassword(username string, newPassword string) (err error) {
	var user *model.User

	if user, err = p.loadUser(username); err != nil {
		return err
	}

	var digest algorithm.Digest

	if digest, err = p.hash.Hash(newPassword); err != nil {
		return err
	}

	user.Password = schema.NewPasswordDigest(digest)

	if err = p.database.UpdateUserPassword(context.Background(), username, user.Password.Encode()); err != nil {
		return err
	}

	return nil
}

// loadUser load user info, returns error if user not exists or is disabled.
func (p *DBUserProvider) loadUser(username string) (*model.User, error) {
	var user model.User

	var err error

	// TODO: see howto inject a real context.
	if user, err = p.database.LoadUser(context.Background(), username); err != nil {
		return nil, err
	}

	if user.Disabled {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

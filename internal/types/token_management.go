package types

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"

	"github.com/bisonschweizag/gws-cli/internal/log"
)

const TokenFileName = "token.yaml"

type TokenStorage struct {
	Token oauth2.Token `yaml:"token"`
}

func GetTokenFilePath() (string, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	tokenDir := filepath.Join(userHomeDir, ConfigDir)
	if err := os.MkdirAll(tokenDir, 0o700); err != nil {
		return "", err
	}

	return filepath.Join(tokenDir, TokenFileName), nil
}

func LoadToken() (*oauth2.Token, error) {
	kr, err := keyring.Get("gws", "token")
	if err == nil {
		var storage TokenStorage
		if err := yaml.Unmarshal([]byte(kr), &storage); err != nil {
			return nil, err
		}
		return &storage.Token, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		// Some other keyring error — fallthrough to file-based fallback
		log.Logf("Keyring read error: %v; falling back to file storage", err)
	}

	tokenPath, err := GetTokenFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // No token yet
		}
		return nil, err
	}

	var storage TokenStorage
	err = yaml.Unmarshal(data, &storage)
	if err != nil {
		return nil, err
	}

	return &storage.Token, nil
}

func SaveToken(token oauth2.Token) error {
	log.Logf("🎟️ Got new Google Access Token (expires: %s)", token.Expiry.Format(time.RFC822))
	// Encode token to YAML once and reuse for both keyring and file storage
	storage := TokenStorage{Token: token}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err := encoder.Encode(storage)
	if err != nil {
		return err
	}

	err = keyring.Set("gws", "token", buf.String())
	if err == nil {
		log.Logf("🎟️ Stored token in OS keyring")
		return nil
	}
	log.Logf("Keyring write error: %v; falling back to file storage", err)

	tokenPath, err := GetTokenFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(tokenPath, buf.Bytes(), 0o600)
}

func SetToken(token oauth2.Token) error {
	// Check if token has changed before saving
	existingToken, err := LoadToken()
	if err != nil {
		return err
	}

	if existingToken != nil && existingToken.AccessToken == token.AccessToken {
		return nil // No change
	}

	return SaveToken(token)
}

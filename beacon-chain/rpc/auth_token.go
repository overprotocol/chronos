package rpc

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/io/file"
)

// Mostly taken from validator/rpc/auth_token.go, with some modifications to fit the context of beacon-chain

func getAuthToken(authTokenPath string) (string, error) {
	if authTokenPath == "" {
		return "", errors.New("auth token path is empty")
	}

	exists, err := file.Exists(authTokenPath, file.Regular)
	if err != nil {
		return "", errors.Wrapf(err, "could not check if file %s exists", authTokenPath)
	}
	if exists {
		f, err := os.Open(filepath.Clean(authTokenPath))
		if err != nil {
			return "", err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Error(err)
			}
		}()
		token, err := readAuthTokenFile(f)
		if err != nil {
			return "", err
		}
		return token, nil
	}
	token, err := api.GenerateRandomHexString()
	if err != nil {
		return "", err
	}

	err = saveAuthToken(authTokenPath, token)
	if err != nil {
		return "", err
	}

	return token, nil
}

func readAuthTokenFile(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	var token string
	// Scan the file and collect lines, excluding empty lines
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	switch len(lines) {
	case 1:
		token = strings.TrimSpace(lines[0])
	default:
		return "", errors.New("Auth token file format has multiple lines, please update the auth token to a single line that is a 256 bit hex string")

	}

	if err := api.ValidateAuthToken(token); err != nil {
		log.WithError(err).Warn(
			"Auth token does not follow our standards and should be regenerated " +
				"by removing the current token file and restarting",
		)
	}
	return token, nil
}

func saveAuthToken(authTokenPath, token string) error {
	bytesBuf := new(bytes.Buffer)
	if _, err := bytesBuf.WriteString(token); err != nil {
		return err
	}
	if _, err := bytesBuf.WriteString("\n"); err != nil {
		return err
	}

	if err := file.MkdirAll(filepath.Dir(authTokenPath)); err != nil {
		return errors.Wrapf(err, "could not create directory %s", filepath.Dir(authTokenPath))
	}
	if err := file.WriteFile(authTokenPath, bytesBuf.Bytes()); err != nil {
		return errors.Wrapf(err, "could not write to file %s", authTokenPath)
	}

	return nil
}

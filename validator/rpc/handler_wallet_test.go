package rpc

import (
	"bytes"
	"context"
	rd "crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"
	"github.com/prysmaticlabs/prysm/v5/async/event"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	"github.com/prysmaticlabs/prysm/v5/crypto/aes"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/crypto/rand"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts/iface"
	mock "github.com/prysmaticlabs/prysm/v5/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v5/validator/client"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager/local"
	"github.com/tyler-smith/go-bip39"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

const strongPass = "29384283xasjasd32%%&*@*#*"

func TestServer_CreateWallet_Local(t *testing.T) {
	ctx := context.Background()
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	opts := []accounts.Option{
		accounts.WithWalletDir(defaultWalletPath),
		accounts.WithKeymanagerType(keymanager.Derived),
		accounts.WithWalletPassword(strongPass),
		accounts.WithSkipMnemonicConfirm(true),
	}
	acc, err := accounts.NewCLIManager(opts...)
	require.NoError(t, err)
	w, err := acc.WalletCreate(ctx)
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.Validator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
		validatorService:      vs,
	}
	request := &CreateWalletRequest{
		Keymanager:     importedKeymanagerKind,
		WalletPassword: strongPass,
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/create", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.CreateWallet(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)

	encryptor := keystorev4.New()
	keystores := make([]string, 3)
	passwords := make([]string, 3)
	for i := 0; i < len(keystores); i++ {
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &keymanager.Keystore{
			Crypto:      cryptoFields,
			ID:          id.String(),
			Version:     encryptor.Version(),
			Pubkey:      pubKey,
			Description: encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
		if i < len(passwords) {
			passwords[i] = strongPass
		}
	}

	importReq := &ImportKeystoresRequest{
		Keystores: keystores,
		Passwords: passwords,
	}

	err = json.NewEncoder(&buf).Encode(importReq)
	require.NoError(t, err)

	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/eth/v1/keystores"), &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.ImportKeystores(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)
	resp := &ImportKeystoresResponse{}
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), resp))
	for _, status := range resp.Data {
		require.Equal(t, keymanager.StatusImported, status.Status)
	}
	keys, err := km.FetchValidatingPublicKeys(ctx)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(keystores))
}

func TestServer_CreateWallet_Local_PasswordTooWeak(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	request := &CreateWalletRequest{
		Keymanager:     importedKeymanagerKind,
		WalletPassword: "", // Weak password, empty string
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/create", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.CreateWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "Password too weak", wr.Body.String())

	request = &CreateWalletRequest{
		Keymanager:     importedKeymanagerKind,
		WalletPassword: "a", // Weak password, too short
	}
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/create", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.CreateWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "Password too weak", wr.Body.String())
}

func TestServer_RecoverWallet_Derived(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             localWalletDir,
	}
	request := &RecoverWalletRequest{
		WalletPassword: strongPass,
		NumAccounts:    0,
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.RecoverWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "Must create at least 1 validator account", wr.Body.String())

	request.NumAccounts = 2
	request.Language = "Swahili"
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.RecoverWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "input not in the list of supported languages", wr.Body.String())

	request.Language = "ENglish"
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.RecoverWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "invalid mnemonic in request", wr.Body.String())

	mnemonicRandomness := make([]byte, 32)
	_, err = rand.NewGenerator().Read(mnemonicRandomness)
	require.NoError(t, err)
	mnemonic, err := bip39.NewMnemonic(mnemonicRandomness)
	require.NoError(t, err)
	request.Mnemonic = mnemonic
	request.Mnemonic25ThWord = " "

	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.RecoverWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "mnemonic 25th word cannot be empty", wr.Body.String())

	request.Mnemonic25ThWord = "outer"
	// Test weak password.
	request.WalletPassword = "123qwe"

	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.RecoverWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "password did not pass validation", wr.Body.String())

	request.WalletPassword = strongPass
	// Create(derived) should fail then test recover.
	reqCreate := &CreateWalletRequest{
		Keymanager:     derivedKeymanagerKind,
		WalletPassword: strongPass,
		NumAccounts:    2,
		Mnemonic:       mnemonic,
	}
	var buff bytes.Buffer
	err = json.NewEncoder(&buff).Encode(reqCreate)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/create", &buff)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.CreateWallet(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "create wallet not supported through web", wr.Body.String())

	// This defer will be the last to execute in this func.
	resetCfgFalse := features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: false,
	})
	defer resetCfgFalse()

	resetCfgTrue := features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: true,
	})
	defer resetCfgTrue()

	// Finally test recover.
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.RecoverWallet(wr, req)

	// Password File should have been written.
	passwordFilePath := filepath.Join(localWalletDir, wallet.DefaultWalletPasswordFile)
	exists, err := file.Exists(passwordFilePath, file.Regular)
	require.NoError(t, err, "could not check if password file exists")
	assert.Equal(t, true, exists)

	// Attempting to write again should trigger an error.
	err = writeWalletPasswordToDisk(localWalletDir, "somepassword")
	require.ErrorContains(t, "cannot write wallet password file as it already exists", err)

}

func TestServer_ValidateKeystores_FailedPreconditions(t *testing.T) {
	strongPass := "29384283xasjasd32%%&*@*#*"
	ss := &Server{}
	request := &ValidateKeystoresRequest{}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	ss.ValidateKeystores(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	assert.StringContains(t, "Password required for keystores", wr.Body.String())

	request = &ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
	}
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	ss.ValidateKeystores(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	assert.StringContains(t, "No keystores included in request", wr.Body.String())

	request = &ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
		Keystores:         []string{"badjson"},
	}
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	ss.ValidateKeystores(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	assert.StringContains(t, "Not a valid EIP-2335 keystore", wr.Body.String())
}

func TestServer_ValidateKeystores_OK(t *testing.T) {
	strongPass := "29384283xasjasd32%%&*@*#*"
	ss := &Server{}

	// Create 3 keystores with the strong password.
	encryptor := keystorev4.New()
	keystores := make([]string, 3)
	pubKeys := make([][]byte, 3)
	for i := 0; i < len(keystores); i++ {
		privKey, err := bls.RandKey()
		require.NoError(t, err)
		pubKey := fmt.Sprintf("%x", privKey.PublicKey().Marshal())
		id, err := uuid.NewRandom()
		require.NoError(t, err)
		cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), strongPass)
		require.NoError(t, err)
		item := &keymanager.Keystore{
			Crypto:      cryptoFields,
			ID:          id.String(),
			Version:     encryptor.Version(),
			Pubkey:      pubKey,
			Description: encryptor.Name(),
		}
		encodedFile, err := json.MarshalIndent(item, "", "\t")
		require.NoError(t, err)
		keystores[i] = string(encodedFile)
		pubKeys[i] = privKey.PublicKey().Marshal()
	}

	// Validate the keystores and ensure no error.
	request := &ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
		Keystores:         keystores,
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	ss.ValidateKeystores(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)

	// Check that using a different password will return an error.
	request = &ValidateKeystoresRequest{
		KeystoresPassword: "badpassword",
		Keystores:         keystores,
	}
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	ss.ValidateKeystores(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "is incorrect", wr.Body.String())

	// Add a new keystore that was encrypted with a different password and expect
	// a failure from the function.
	differentPassword := "differentkeystorepass"
	privKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := "somepubkey"
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	cryptoFields, err := encryptor.Encrypt(privKey.Marshal(), differentPassword)
	require.NoError(t, err)
	item := &keymanager.Keystore{
		Crypto:      cryptoFields,
		ID:          id.String(),
		Version:     encryptor.Version(),
		Pubkey:      pubKey,
		Description: encryptor.Name(),
	}
	encodedFile, err := json.MarshalIndent(item, "", "\t")
	keystores = append(keystores, string(encodedFile))
	require.NoError(t, err)
	request = &ValidateKeystoresRequest{
		KeystoresPassword: strongPass,
		Keystores:         keystores,
	}
	err = json.NewEncoder(&buf).Encode(request)
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/recover", &buf)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	ss.ValidateKeystores(wr, req)
	require.NotEqual(t, http.StatusOK, wr.Code)
	require.StringContains(t, "Password for keystore with public key somepubkey is incorrect", wr.Body.String())
}

func TestServer_WalletConfig_NoWalletFound(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/v2/validator/wallet/keystores/validate", nil)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.WalletConfig(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)
	var resp WalletResponse
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), &resp))
	require.DeepEqual(t, resp, WalletResponse{})
}

func TestServer_WalletConfig(t *testing.T) {
	localWalletDir := setupWalletDir(t)
	defaultWalletPath = localWalletDir
	ctx := context.Background()
	s := &Server{
		walletInitializedFeed: new(event.Feed),
		walletDir:             defaultWalletPath,
	}
	// We attempt to create the wallet.
	opts := []accounts.Option{
		accounts.WithWalletDir(defaultWalletPath),
		accounts.WithKeymanagerType(keymanager.Local),
		accounts.WithWalletPassword(strongPass),
		accounts.WithSkipMnemonicConfirm(true),
	}
	acc, err := accounts.NewCLIManager(opts...)
	require.NoError(t, err)
	w, err := acc.WalletCreate(ctx)
	require.NoError(t, err)
	km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
	require.NoError(t, err)
	s.wallet = w
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Wallet: w,
		Validator: &mock.Validator{
			Km: km,
		},
	})
	require.NoError(t, err)
	s.validatorService = vs
	req := httptest.NewRequest(http.MethodGet, "/v2/validator/wallet/keystores/validate", nil)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.WalletConfig(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)
	var resp WalletResponse
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), &resp))

	assert.DeepEqual(t, resp, WalletResponse{
		WalletPath:     localWalletDir,
		KeymanagerKind: importedKeymanagerKind,
	})
}

func TestServer_InitializeWallet_Ok(t *testing.T) {
	ctx := context.Background()
	cipher, err := generateRandomKey()
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: &mock.Validator{},
	})

	if err != nil {
		t.Fatal(err)
	}

	s := &Server{
		useOverNode:           true,
		walletInitializedFeed: new(event.Feed),
		cipherKey:             cipher,
		validatorService:      vs,
	}
	password := "testpassword"
	encryptedPassword, err := aes.Encrypt(s.cipherKey, []byte(password))

	require.NoError(t, err)

	// Test case 1. Working case.
	testPath := "./testpath"

	// new path
	req1Body := &InitializeWalletRequest{
		WalletDir: testPath,
		Password:  hexutil.Encode(encryptedPassword),
	}
	var buf1 bytes.Buffer
	err = json.NewEncoder(&buf1).Encode(req1Body)
	require.NoError(t, err)

	req1 := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/initialize-wallet", &buf1)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.InitializeWallet(wr, req1)
	require.Equal(t, http.StatusOK, wr.Code)
	resp1 := &InitializeWalletResponse{}
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), resp1))
	assert.Equal(t, testPath, resp1.WalletDir)

	s.wallet = nil
	s.walletInitialized = false

	// exist and normal path
	req2Body := &InitializeWalletRequest{
		WalletDir: testPath,
		Password:  hexutil.Encode(encryptedPassword),
	}
	var buf2 bytes.Buffer
	err = json.NewEncoder(&buf2).Encode(req2Body)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/initialize-wallet", &buf2)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.InitializeWallet(wr, req2)
	require.Equal(t, http.StatusOK, wr.Code)
	resp2 := &InitializeWalletResponse{}
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), resp2))
	assert.Equal(t, testPath, resp2.WalletDir)
}

func TestServer_InitializeWallet_WalletAlreadyOpened(t *testing.T) {
	ctx := context.Background()
	cipher, err := generateRandomKey()
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: &mock.Validator{},
	})

	if err != nil {
		t.Fatal(err)
	}

	s := &Server{
		useOverNode:           true,
		walletInitializedFeed: new(event.Feed),
		cipherKey:             cipher,
		validatorService:      vs,
	}
	password := "testpassword"
	encryptedPassword, err := aes.Encrypt(s.cipherKey, []byte(password))

	require.NoError(t, err)

	testPath := "./testpath"
	req1Body := &InitializeWalletRequest{
		WalletDir: testPath,
		Password:  hexutil.Encode(encryptedPassword),
	}
	var buf1 bytes.Buffer
	err = json.NewEncoder(&buf1).Encode(req1Body)
	require.NoError(t, err)

	req1 := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/initialize-wallet", &buf1)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.InitializeWallet(wr, req1)
	require.Equal(t, http.StatusOK, wr.Code)
	resp1 := &InitializeWalletResponse{}
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), resp1))
	assert.Equal(t, testPath, resp1.WalletDir)

	// try to re-open wallet
	testPath = "./testpath2"
	req2Body := &InitializeWalletRequest{
		WalletDir: testPath,
		Password:  hexutil.Encode(encryptedPassword),
	}
	var buf2 bytes.Buffer
	err = json.NewEncoder(&buf2).Encode(req2Body)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/initialize-wallet", &buf2)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.InitializeWallet(wr, req2)
	require.Equal(t, http.StatusConflict, wr.Code)
	require.StringContains(t, "Wallet is Already Opened", string(wr.Body.Bytes()))
}

func TestServer_InitializeWallet_WrongCipherKey(t *testing.T) {
	ctx := context.Background()
	cipher, err := generateRandomKey()
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: &mock.Validator{},
	})

	if err != nil {
		t.Fatal(err)
	}

	s := &Server{
		useOverNode:           true,
		walletInitializedFeed: new(event.Feed),
		cipherKey:             cipher,
		validatorService:      vs,
	}
	password := "testpassword"
	encryptedPassword, err := aes.Encrypt(s.cipherKey, []byte(password))

	require.NoError(t, err)

	testPath := "./testpath"
	req1Body := &InitializeWalletRequest{
		WalletDir: testPath,
		Password:  hexutil.Encode(encryptedPassword),
	}
	var buf1 bytes.Buffer
	err = json.NewEncoder(&buf1).Encode(req1Body)
	require.NoError(t, err)

	req1 := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/initialize-wallet", &buf1)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.InitializeWallet(wr, req1)
	require.Equal(t, http.StatusOK, wr.Code)
	resp1 := &InitializeWalletResponse{}
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), resp1))
	assert.Equal(t, testPath, resp1.WalletDir)

	s.wallet = nil
	s.walletInitialized = false

	// try to add wrong encrypted password
	wrongCipher, err := generateRandomKey()
	require.NoError(t, err)

	wrongEncryptedPassword, err := aes.Encrypt(wrongCipher, []byte(password))
	require.NoError(t, err)

	req2Body := &InitializeWalletRequest{
		WalletDir: testPath,
		Password:  hexutil.Encode(wrongEncryptedPassword),
	}
	var buf2 bytes.Buffer
	err = json.NewEncoder(&buf2).Encode(req2Body)
	require.NoError(t, err)

	req2 := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/initialize-wallet", &buf2)
	wr = httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.InitializeWallet(wr, req2)
	require.Equal(t, http.StatusBadRequest, wr.Code)
	require.StringContains(t, "Could not decrypt password", string(wr.Body.Bytes()))
}

func TestServer_InitializeWallet_ThreadSafe(t *testing.T) {
	ctx := context.Background()
	cipher, err := generateRandomKey()
	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: &mock.Validator{},
	})

	if err != nil {
		t.Fatal(err)
	}

	s := &Server{
		useOverNode:           true,
		walletInitializedFeed: new(event.Feed),
		cipherKey:             cipher,
		validatorService:      vs,
	}
	password := "testpassword"
	encryptedPassword, err := aes.Encrypt(s.cipherKey, []byte(password))

	require.NoError(t, err)

	testPath := "./testpath"

	reqBody := &InitializeWalletRequest{
		WalletDir: testPath,
		Password:  hexutil.Encode(encryptedPassword),
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/initialize-wallet", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.InitializeWallet(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)
	resp := &InitializeWalletResponse{}
	require.NoError(t, json.Unmarshal(wr.Body.Bytes(), resp))
	assert.Equal(t, testPath, resp.WalletDir)

	// Create a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Create a number of goroutines that will call InitializeWallet concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wr := httptest.NewRecorder()
			wr.Body = &bytes.Buffer{}
			s.InitializeWallet(wr, req)
			if wr.Code != http.StatusOK {
				require.StringContains(t, "Wallet is Already Opened", string(wr.Body.Bytes()))
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Check if the wallet, walletInitialized, and walletDir fields have the expected values
	require.Equal(t, true, s.walletInitialized)
	require.Equal(t, testPath, s.walletDir)
}

func TestServer_ChangePassword_Ok(t *testing.T) {
	ctx := context.Background()
	cipher, err := generateRandomKey()
	require.NoError(t, err)
	password := "testpassword"
	encryptedPassword, err := aes.Encrypt(cipher, []byte(password))
	require.NoError(t, err)

	walletDir := setupWalletDir(t)
	w := wallet.New(&wallet.Config{
		WalletDir:      walletDir,
		KeymanagerKind: keymanager.Local,
		WalletPassword: password,
	})

	km, err := local.NewKeymanager(ctx, &local.SetupConfig{
		Wallet:           w,
		ListenForChanges: true,
	})
	require.NoError(t, err)
	keystores := createRandomKeystore(t, password)
	_, err = km.ImportKeystores(ctx, []*keymanager.Keystore{keystores}, []string{password})

	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: &mock.Validator{
			Km: km,
		},
	})
	require.NoError(t, err)

	if err != nil {
		t.Fatal(err)
	}

	// Create a Server instance
	s := &Server{
		useOverNode:           true,
		walletInitializedFeed: new(event.Feed),
		cipherKey:             cipher,
		validatorService:      vs,
		wallet:                w,
		walletInitialized:     true,
	}

	newPassword := "newPassword"
	encryptedNewPassword, err := aes.Encrypt(cipher, []byte(newPassword))
	require.NoError(t, err)

	reqBody := &ChangePasswordRequest{
		Password:    hexutil.Encode(encryptedPassword),
		NewPassword: hexutil.Encode(encryptedNewPassword),
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/change-password", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.ChangePassword(wr, req)
	require.Equal(t, http.StatusOK, wr.Code)
}

func TestServer_ChangePassword_WrongPassword(t *testing.T) {
	ctx := context.Background()
	cipher, err := generateRandomKey()
	require.NoError(t, err)
	password := "testpassword"

	walletDir := setupWalletDir(t)
	w := wallet.New(&wallet.Config{
		WalletDir:      walletDir,
		KeymanagerKind: keymanager.Local,
		WalletPassword: password,
	})

	km, err := local.NewKeymanager(ctx, &local.SetupConfig{
		Wallet:           w,
		ListenForChanges: true,
	})
	require.NoError(t, err)
	keystores := createRandomKeystore(t, password)
	_, err = km.ImportKeystores(ctx, []*keymanager.Keystore{keystores}, []string{password})

	require.NoError(t, err)
	vs, err := client.NewValidatorService(ctx, &client.Config{
		Validator: &mock.Validator{
			Km: km,
		},
	})
	require.NoError(t, err)

	if err != nil {
		t.Fatal(err)
	}

	// Create a Server instance
	s := &Server{
		useOverNode:           true,
		walletInitializedFeed: new(event.Feed),
		cipherKey:             cipher,
		validatorService:      vs,
		wallet:                w,
		walletInitialized:     true,
	}

	newPassword := "newPassword"
	encryptedNewPassword, err := aes.Encrypt(cipher, []byte(newPassword))
	require.NoError(t, err)

	wrongPassword := "wrongPassword"
	encryptedWrongPassword, err := aes.Encrypt(cipher, []byte(wrongPassword))
	require.NoError(t, err)

	reqBody := &ChangePasswordRequest{
		Password:    hexutil.Encode(encryptedWrongPassword),
		NewPassword: hexutil.Encode(encryptedNewPassword),
	}
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(reqBody)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/change-password", &buf)
	wr := httptest.NewRecorder()
	wr.Body = &bytes.Buffer{}
	s.ChangePassword(wr, req)
	require.Equal(t, http.StatusBadRequest, wr.Code)
	require.StringContains(t, "Old password is not correct", string(wr.Body.Bytes()))
}

func Test_writeWalletPasswordToDisk(t *testing.T) {
	walletDir := setupWalletDir(t)
	resetCfg := features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: false,
	})
	defer resetCfg()
	err := writeWalletPasswordToDisk(walletDir, "somepassword")
	require.NoError(t, err)

	// Expected a silent failure if the feature flag is not enabled.
	passwordFilePath := filepath.Join(walletDir, wallet.DefaultWalletPasswordFile)
	exists, err := file.Exists(passwordFilePath, file.Regular)
	require.NoError(t, err, "could not check if password file exists")
	assert.Equal(t, false, exists, "password file should not exist")
	resetCfg = features.InitWithReset(&features.Flags{
		WriteWalletPasswordOnWebOnboarding: true,
	})
	defer resetCfg()
	err = writeWalletPasswordToDisk(walletDir, "somepassword")
	require.NoError(t, err)

	// File should have been written.
	exists, err = file.Exists(passwordFilePath, file.Regular)
	require.NoError(t, err, "could not check if password file exists")
	assert.Equal(t, true, exists, "password file should exist")

	// Attempting to write again should trigger an error.
	err = writeWalletPasswordToDisk(walletDir, "somepassword")
	require.NotNil(t, err)
}

func createRandomKeystore(t testing.TB, password string) *keymanager.Keystore {
	encryptor := keystorev4.New()
	id, err := uuid.NewRandom()
	require.NoError(t, err)
	validatingKey, err := bls.RandKey()
	require.NoError(t, err)
	pubKey := validatingKey.PublicKey().Marshal()
	cryptoFields, err := encryptor.Encrypt(validatingKey.Marshal(), password)
	require.NoError(t, err)
	return &keymanager.Keystore{
		Crypto:      cryptoFields,
		Pubkey:      fmt.Sprintf("%x", pubKey),
		ID:          id.String(),
		Version:     encryptor.Version(),
		Description: encryptor.Name(),
	}
}

func generateRandomKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rd.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func TestServer_ChangePassword_PreventBruteForce(t *testing.T) {
	t.Run("LockoutAfterMultipleFailedAttempts", func(t *testing.T) {
		// 1. Set up a Server with a known correct old password.
		ctx := context.Background()
		cipher, err := generateRandomKey()
		require.NoError(t, err)
		oldPassword := "correct-old-password"

		walletDir := setupWalletDir(t)
		w := wallet.New(&wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: oldPassword,
		})
		km, err := local.NewKeymanager(ctx, &local.SetupConfig{
			Wallet:           w,
			ListenForChanges: true,
		})
		require.NoError(t, err)

		// Import one random keystore to fully initialize the wallet.
		ks := createRandomKeystore(t, oldPassword)
		_, err = km.ImportKeystores(ctx, []*keymanager.Keystore{ks}, []string{oldPassword})
		require.NoError(t, err)

		vs, err := client.NewValidatorService(ctx, &client.Config{
			Validator: &mock.Validator{Km: km},
			Wallet:    w,
		})
		require.NoError(t, err)

		s := &Server{
			useOverNode:           true, // must be true to enable ChangePassword
			walletInitializedFeed: new(event.Feed),
			cipherKey:             cipher,
			validatorService:      vs,
			wallet:                w,
			walletInitialized:     true,
		}

		// 2. Encrypt a WRONG old password for repeated attempts.
		wrongPassword := "wrong-old-password"
		encryptedWrongPassword, err := aes.Encrypt(cipher, []byte(wrongPassword))
		require.NoError(t, err)
		newPassword := "someNewPassword"
		encryptedNewPassword, err := aes.Encrypt(cipher, []byte(newPassword))
		require.NoError(t, err)

		// 3. Try multiple times with the wrong old password.
		maxAttempts := 5 // or 5, depending on your actual code’s logic
		for i := 1; i <= maxAttempts; i++ {

			reqBody := &ChangePasswordRequest{
				Password:    hexutil.Encode(encryptedWrongPassword),
				NewPassword: hexutil.Encode(encryptedNewPassword),
			}
			var buf bytes.Buffer
			require.NoError(t, json.NewEncoder(&buf).Encode(reqBody))

			req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/change-password", &buf)
			wr := httptest.NewRecorder()
			wr.Body = &bytes.Buffer{}
			s.ChangePassword(wr, req)

			if i < maxAttempts {
				// Expect a 400 with "Old password is not correct".
				require.Equal(t, http.StatusBadRequest, wr.Code)
				require.StringContains(t, "Old password is not correct", wr.Body.String())
			} else {
				// On the final attempt, we expect the server to lock out the user (429 or similar).
				require.Equal(t, http.StatusBadRequest, wr.Code)
				require.StringContains(t, "Too many failed attempts", wr.Body.String())
			}
		}
	})

	t.Run("CannotChangePasswordWhileLockedOut", func(t *testing.T) {
		// 1. Recreate server with correct old password.
		ctx := context.Background()
		cipher, err := generateRandomKey()
		require.NoError(t, err)
		oldPassword := "correct-old-password"

		walletDir := setupWalletDir(t)
		w := wallet.New(&wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: oldPassword,
		})
		km, err := local.NewKeymanager(ctx, &local.SetupConfig{
			Wallet:           w,
			ListenForChanges: true,
		})
		require.NoError(t, err)
		ks := createRandomKeystore(t, oldPassword)
		_, err = km.ImportKeystores(ctx, []*keymanager.Keystore{ks}, []string{oldPassword})
		require.NoError(t, err)

		vs, err := client.NewValidatorService(ctx, &client.Config{
			Validator: &mock.Validator{Km: km},
			Wallet:    w,
		})
		require.NoError(t, err)

		s := &Server{
			useOverNode:           true,
			walletInitializedFeed: new(event.Feed),
			cipherKey:             cipher,
			validatorService:      vs,
			wallet:                w,
			walletInitialized:     true,
		}

		// 2. Force the server into a "locked-out" state.
		//    In your actual code, s.lockoutUntil might be the field used.
		s.failedPasswordAttempts = 999 // big number
		// (In real usage, you'd set s.lockoutUntil to time.Now().Add(...) )

		wrongPassword := "wrong-old-password"
		encryptedWrongPassword, err := aes.Encrypt(cipher, []byte(wrongPassword))
		require.NoError(t, err)
		encryptedNewPassword, err := aes.Encrypt(cipher, []byte("newValidPassword"))
		require.NoError(t, err)

		reqBody := &ChangePasswordRequest{
			Password:    hexutil.Encode(encryptedWrongPassword),
			NewPassword: hexutil.Encode(encryptedNewPassword),
		}
		var buf bytes.Buffer
		require.NoError(t, json.NewEncoder(&buf).Encode(reqBody))

		req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/change-password", &buf)
		wr := httptest.NewRecorder()
		wr.Body = &bytes.Buffer{}
		s.ChangePassword(wr, req)

		req = httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/change-password", &buf)
		wr = httptest.NewRecorder()
		wr.Body = &bytes.Buffer{}
		s.ChangePassword(wr, req)

		// Expect locked-out response (429 or 400 with a lockout message).
		// Adjust to match your actual code’s status + message.
		require.Equal(t, http.StatusBadRequest, wr.Code)
		require.StringContains(t, "Too many failed attempts", wr.Body.String())
	})

	t.Run("ResetsFailuresOnSuccessfulOldPassword", func(t *testing.T) {
		// 1. Set up a server with a known correct old password.
		ctx := context.Background()
		cipher, err := generateRandomKey()
		require.NoError(t, err)
		oldPassword := "correct-old-password"

		walletDir := setupWalletDir(t)
		w := wallet.New(&wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: oldPassword,
		})
		km, err := local.NewKeymanager(ctx, &local.SetupConfig{
			Wallet:           w,
			ListenForChanges: true,
		})
		require.NoError(t, err)
		ks := createRandomKeystore(t, oldPassword)
		_, err = km.ImportKeystores(ctx, []*keymanager.Keystore{ks}, []string{oldPassword})
		require.NoError(t, err)

		vs, err := client.NewValidatorService(ctx, &client.Config{
			Validator: &mock.Validator{Km: km},
			Wallet:    w,
		})
		require.NoError(t, err)

		s := &Server{
			useOverNode:            true,
			walletInitializedFeed:  new(event.Feed),
			cipherKey:              cipher,
			validatorService:       vs,
			wallet:                 w,
			walletInitialized:      true,
			failedPasswordAttempts: 2, // Some prior failures
		}

		// 2. Encrypt the correct old password and a valid new password.
		encryptedOldPassword, err := aes.Encrypt(cipher, []byte(oldPassword))
		require.NoError(t, err)
		encryptedNewPassword, err := aes.Encrypt(cipher, []byte("brandNewStrongPassword"))
		require.NoError(t, err)

		reqBody := &ChangePasswordRequest{
			Password:    hexutil.Encode(encryptedOldPassword),
			NewPassword: hexutil.Encode(encryptedNewPassword),
		}
		var buf bytes.Buffer
		require.NoError(t, json.NewEncoder(&buf).Encode(reqBody))

		req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/change-password", &buf)
		wr := httptest.NewRecorder()
		wr.Body = &bytes.Buffer{}
		s.ChangePassword(wr, req)

		// Expect success (200 OK).
		require.Equal(t, http.StatusOK, wr.Code)
		require.StringContains(t, "Password changed successfully", wr.Body.String())

		// Confirm the failure count was reset to 0.
		assert.Equal(t, 0, s.failedPasswordAttempts, "failedPasswordAttempts should be reset on success")
	})

	t.Run("ShortNewPasswordFailsValidation", func(t *testing.T) {
		// 1. Set up a server with the correct old password.
		ctx := context.Background()
		cipher, err := generateRandomKey()
		require.NoError(t, err)
		oldPassword := "my-super-secret-oldpass"

		walletDir := setupWalletDir(t)
		w := wallet.New(&wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanager.Local,
			WalletPassword: oldPassword,
		})
		km, err := local.NewKeymanager(ctx, &local.SetupConfig{
			Wallet:           w,
			ListenForChanges: true,
		})
		require.NoError(t, err)
		ks := createRandomKeystore(t, oldPassword)
		_, err = km.ImportKeystores(ctx, []*keymanager.Keystore{ks}, []string{oldPassword})
		require.NoError(t, err)

		vs, err := client.NewValidatorService(ctx, &client.Config{
			Validator: &mock.Validator{Km: km},
			Wallet:    w,
		})
		require.NoError(t, err)

		s := &Server{
			useOverNode:           true,
			walletInitializedFeed: new(event.Feed),
			cipherKey:             cipher,
			validatorService:      vs,
			wallet:                w,
			walletInitialized:     true,
		}

		// 2. Provide correct old password but an obviously too-short new password
		encryptedOldPassword, err := aes.Encrypt(cipher, []byte(oldPassword))
		require.NoError(t, err)
		encryptedShortPassword, err := aes.Encrypt(cipher, []byte("12")) // fails ValidatePasswordInput?
		require.NoError(t, err)

		reqBody := &ChangePasswordRequest{
			Password:    hexutil.Encode(encryptedOldPassword),
			NewPassword: hexutil.Encode(encryptedShortPassword),
		}
		var buf bytes.Buffer
		require.NoError(t, json.NewEncoder(&buf).Encode(reqBody))

		req := httptest.NewRequest(http.MethodPost, "/v2/validator/wallet/change-password", &buf)
		wr := httptest.NewRecorder()
		wr.Body = &bytes.Buffer{}
		s.ChangePassword(wr, req)

		// Expect a 400 with a "Password too weak" or similar message
		// (depends on how ValidatePasswordInput returns its errors).
		require.Equal(t, http.StatusBadRequest, wr.Code)
		require.StringContains(t, "New password does not meet criteria", wr.Body.String())
	})
}

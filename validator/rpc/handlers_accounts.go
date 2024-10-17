package rpc

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/pagination"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/rpc/eth/shared"
	"github.com/prysmaticlabs/prysm/v5/cmd"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/crypto/aes"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts"
	"github.com/prysmaticlabs/prysm/v5/validator/accounts/petnames"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager/local"
	keystorev4 "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4"
)

var tempPassword = "temp-password"

// ListAccounts allows retrieval of validating keys and their petnames
// for a user's wallet via RPC.
func (s *Server) ListAccounts(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.web.accounts.ListAccounts")
	defer span.End()
	if s.validatorService == nil {
		httputil.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	if !s.walletInitialized {
		httputil.HandleError(w, "Prysm Wallet not initialized. Please create a new wallet.", http.StatusServiceUnavailable)
		return
	}
	pageSize := r.URL.Query().Get("page_size")
	var ps int64
	if pageSize != "" {
		psi, err := strconv.ParseInt(pageSize, 10, 32)
		if err != nil {
			httputil.HandleError(w, errors.Wrap(err, "Failed to parse page_size").Error(), http.StatusBadRequest)
			return
		}
		ps = psi
	}
	pageToken := r.URL.Query().Get("page_token")
	publicKeys := r.URL.Query()["public_keys"]
	pubkeys := make([][]byte, len(publicKeys))
	for i, key := range publicKeys {
		k, ok := shared.ValidateHex(w, fmt.Sprintf("PublicKeys[%d]", i), key, fieldparams.BLSPubkeyLength)
		if !ok {
			return
		}
		pubkeys[i] = bytesutil.SafeCopyBytes(k)
	}
	if int(ps) > cmd.Get().MaxRPCPageSize {
		httputil.HandleError(w, fmt.Sprintf("Requested page size %d can not be greater than max size %d",
			ps, cmd.Get().MaxRPCPageSize), http.StatusBadRequest)
		return
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	keys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		httputil.HandleError(w, errors.Errorf("Could not retrieve public keys: %v", err).Error(), http.StatusInternalServerError)
		return
	}
	accs := make([]*Account, len(keys))
	for i := 0; i < len(keys); i++ {
		accs[i] = &Account{
			ValidatingPublicKey: hexutil.Encode(keys[i][:]),
			AccountName:         petnames.DeterministicName(keys[i][:], "-"),
		}
		if s.wallet.KeymanagerKind() == keymanager.Derived {
			accs[i].DerivationPath = fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, i)
		}
	}
	if r.URL.Query().Get("all") == "true" {
		httputil.WriteJson(w, &ListAccountsResponse{
			Accounts:      accs,
			TotalSize:     int32(len(keys)),
			NextPageToken: "",
		})
		return
	}
	start, end, nextPageToken, err := pagination.StartAndEndPage(pageToken, int(ps), len(keys))
	if err != nil {
		httputil.HandleError(w, fmt.Errorf("Could not paginate results: %w",
			err).Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJson(w, &ListAccountsResponse{
		Accounts:      accs[start:end],
		TotalSize:     int32(len(keys)),
		NextPageToken: nextPageToken,
	})
}

// BackupAccounts creates a zip file containing EIP-2335 keystores for the user's
// specified public keys by encrypting them with the specified password.
func (s *Server) BackupAccounts(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.web.accounts.ListAccounts")
	defer span.End()
	if s.validatorService == nil {
		httputil.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	if !s.walletInitialized {
		httputil.HandleError(w, "Prysm Wallet not initialized. Please create a new wallet.", http.StatusServiceUnavailable)
		return
	}

	var req BackupAccountsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.PublicKeys == nil || len(req.PublicKeys) < 1 {
		httputil.HandleError(w, "No public keys specified to backup", http.StatusBadRequest)
		return
	}
	if req.BackupPassword == "" {
		httputil.HandleError(w, "Backup password cannot be empty", http.StatusBadRequest)
		return
	}

	km, err := s.validatorService.Keymanager()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pubKeys := make([]bls.PublicKey, len(req.PublicKeys))
	for i, key := range req.PublicKeys {
		byteskey, ok := shared.ValidateHex(w, "pubkey", key, fieldparams.BLSPubkeyLength)
		if !ok {
			return
		}
		pubKey, err := bls.PublicKeyFromBytes(byteskey)
		if err != nil {
			httputil.HandleError(w, errors.Wrap(err, fmt.Sprintf("%s Not a valid BLS public key", key)).Error(), http.StatusBadRequest)
			return
		}
		pubKeys[i] = pubKey
	}

	var keystoresToBackup []*keymanager.Keystore
	switch km := km.(type) {
	case *local.Keymanager:
		keystoresToBackup, err = km.ExtractKeystores(ctx, pubKeys, req.BackupPassword)
		if err != nil {
			httputil.HandleError(w, errors.Wrap(err, "Could not backup accounts for local keymanager").Error(), http.StatusInternalServerError)
			return
		}
	case *derived.Keymanager:
		keystoresToBackup, err = km.ExtractKeystores(ctx, pubKeys, req.BackupPassword)
		if err != nil {
			httputil.HandleError(w, errors.Wrap(err, "Could not backup accounts for derived keymanager").Error(), http.StatusInternalServerError)
			return
		}
	default:
		httputil.HandleError(w, "Only HD or IMPORTED wallets can backup accounts", http.StatusBadRequest)
		return
	}
	if len(keystoresToBackup) == 0 {
		httputil.HandleError(w, "No keystores to backup", http.StatusBadRequest)
		return
	}

	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	for i, k := range keystoresToBackup {
		encodedFile, err := json.MarshalIndent(k, "", "\t")
		if err != nil {
			if err := writer.Close(); err != nil {
				log.WithError(err).Error("Could not close zip file after writing")
			}
			httputil.HandleError(w, "could not marshal keystore to JSON file", http.StatusInternalServerError)
			return
		}
		f, err := writer.Create(fmt.Sprintf("keystore-%d.json", i))
		if err != nil {
			if err := writer.Close(); err != nil {
				log.WithError(err).Error("Could not close zip file after writing")
			}
			httputil.HandleError(w, "Could not write keystore file to zip", http.StatusInternalServerError)
			return
		}
		if _, err = f.Write(encodedFile); err != nil {
			if err := writer.Close(); err != nil {
				log.WithError(err).Error("Could not close zip file after writing")
			}
			httputil.HandleError(w, "Could not write keystore file contents", http.StatusBadRequest)
			return
		}
	}
	if err := writer.Close(); err != nil {
		log.WithError(err).Error("Could not close zip file after writing")
	}
	httputil.WriteJson(w, &BackupAccountsResponse{
		ZipFile: base64.StdEncoding.EncodeToString(buf.Bytes()), // convert to base64 string for processing
	})
}

// VoluntaryExit performs a voluntary exit for the validator keys specified in a request.
func (s *Server) VoluntaryExit(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "validator.web.accounts.VoluntaryExit")
	defer span.End()
	if s.validatorService == nil {
		httputil.HandleError(w, "Validator service not ready.", http.StatusServiceUnavailable)
		return
	}
	if !s.walletInitialized {
		httputil.HandleError(w, "Prysm Wallet not initialized. Please create a new wallet.", http.StatusServiceUnavailable)
		return
	}
	var req VoluntaryExitRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.PublicKeys) == 0 {
		httputil.HandleError(w, "No public keys specified to delete", http.StatusBadRequest)
		return
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		httputil.HandleError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pubKeys := make([][]byte, len(req.PublicKeys))
	for i, key := range req.PublicKeys {
		byteskey, ok := shared.ValidateHex(w, "pubkey", key, fieldparams.BLSPubkeyLength)
		if !ok {
			return
		}
		pubKeys[i] = byteskey
	}
	cfg := accounts.PerformExitCfg{
		ValidatorClient:  s.beaconNodeValidatorClient,
		NodeClient:       s.nodeClient,
		Keymanager:       km,
		RawPubKeys:       pubKeys,
		FormattedPubKeys: req.PublicKeys,
	}
	rawExitedKeys, _, err := accounts.PerformVoluntaryExit(ctx, cfg)
	if err != nil {
		httputil.HandleError(w, errors.Wrap(err, "Could not perform voluntary exit").Error(), http.StatusInternalServerError)
		return
	}
	httputil.WriteJson(w, &VoluntaryExitResponse{
		ExitedKeys: rawExitedKeys,
	})
}

// CreateDepositDataList creates DepositData list with given request.
// NOTE: Validator client does not store these values.
// Only for OverNode
func (s *Server) CreateDepositDataList(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "OverNode.CreateDepositDataList")
	defer span.End()

	if !s.useOverNode {
		log.Debug("CreateDepositDataList was called when over node flag disabled")
		httputil.HandleError(w, "Only available in over-node flag enabled", http.StatusNotFound)
		return
	}
	if s.validatorService == nil {
		log.Debug("CreateDepositDataList was called when validator service is not opened")
		httputil.HandleError(w, "Validator Service is Not Opened", http.StatusServiceUnavailable)
		return
	}
	if s.wallet == nil {
		log.Debug("CreateDepositDataList was called when wallet is not opened")
		httputil.HandleError(w, "Wallet is Not Opened", http.StatusServiceUnavailable)
		return
	}

	var req CreateDepositDataListRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.DepositDataInputs) == 0 {
		log.Debug("CreateDepositDataList was called with empty Deposit Data")
		httputil.HandleError(w, "Deposit Data Keys is Empty", http.StatusBadRequest)
		return
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		log.WithError(err).Error("Could not get keymanager")
		httputil.HandleError(w, "Could not get keymanager", http.StatusInternalServerError)
		return
	}

	datas := make([]*DepositDataResponse, len(req.DepositDataInputs))

	for i, key := range req.DepositDataInputs {
		pubKey, err := bls.PublicKeyFromBytes(key.Pubkey)
		if err != nil {
			log.WithError(err).Error("Could not parse public key")
			httputil.HandleError(w, "Could not parse public key", http.StatusInternalServerError)
			return
		}
		amountGwei, err := strconv.ParseUint(key.AmountGwei, 10, 64)
		if err != nil {
			log.WithError(err).Error("Could not parse amount gwei")
			httputil.HandleError(w, "Could not parse amount gwei", http.StatusInternalServerError)
			return
		}
		dd, err := createDepositData(ctx, pubKey, key.WithdrawKey, amountGwei, km.Sign)
		if err != nil {
			log.WithError(err).Error("Could not create deposit data")
			httputil.HandleError(w, "Could not create deposit data", http.StatusInternalServerError)
			return
		}
		datas[i] = dd
	}

	httputil.WriteJson(w, &CreateDepositDataListResponse{
		DepositDataList: datas,
	})
}

// ImportAccountsWithPrivateKey import accounts to keystore.
// private keys are encrypted with cipher key provided by OverNode.
// Only For OverNode.
func (s *Server) ImportAccountsWithPrivateKey(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.StartSpan(r.Context(), "OverNode.ImportAccountsWithPrivateKey")
	defer span.End()

	s.rpcMutex.Lock()
	defer s.rpcMutex.Unlock()
	if !s.useOverNode {
		log.Debug("ImportAccountsWithPrivateKey was called when over node flag disabled")
		httputil.HandleError(w, "Only available in over-node flag enabled", http.StatusNotFound)
		return
	}
	if s.validatorService == nil {
		log.Debug("ImportAccountsWithPrivateKey was called when validator service is not opened")
		httputil.HandleError(w, "Validator Service is Not Opened", http.StatusServiceUnavailable)
		return
	}
	if s.wallet == nil {
		log.Debug("ImportAccountsWithPrivateKey was called when wallet is not opened")
		httputil.HandleError(w, "Wallet is Not Opened", http.StatusServiceUnavailable)
		return
	}
	if !s.walletInitialized {
		httputil.HandleError(w, "Wallet is not initialized", http.StatusBadRequest)
		return
	}

	var req ImportAccountsWithPrivateKeyRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	switch {
	case errors.Is(err, io.EOF):
		httputil.HandleError(w, "No data submitted", http.StatusBadRequest)
		return
	case err != nil:
		httputil.HandleError(w, "Could not decode request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.PrivateKeys) == 0 {
		log.Debug("ImportAccounts was called with empty Deposit Data")
		httputil.HandleError(w, "Deposit Data Keys is Empty", http.StatusBadRequest)
		return
	}
	km, err := s.validatorService.Keymanager()
	if err != nil {
		log.WithError(err).Error("Could not get keymanager")
		httputil.HandleError(w, "Could not get keymanager", http.StatusInternalServerError)
		return
	}

	importer, ok := km.(keymanager.Importer)
	if !ok {
		log.WithError(err).Error("Keymanager cannot import local keys")
		httputil.HandleError(w, "Keymanager cannot import local keys", http.StatusInternalServerError)
		return
	}

	// Decrypt PrivateKeys with cipherKey
	secretKeys := make([]bls.SecretKey, 0)
	for _, encodedPrivateKey := range req.PrivateKeys {
		privateKey, err := hexutil.Decode(encodedPrivateKey)
		if err != nil {
			log.Debug("Could not decrypt private key", err)
			httputil.HandleError(w, "Could not decrypt private key", http.StatusBadRequest)
			return
		}
		decryptedPrivateKey, err := aes.Decrypt(s.cipherKey, privateKey)
		if err != nil {
			log.Debug("Could not decrypt private key", err)
			httputil.HandleError(w, "Could not decrypt private key", http.StatusBadRequest)
			return
		}

		secretKey, err := bls.SecretKeyFromBytes(decryptedPrivateKey)
		if err != nil {
			log.Debug("Could not decrypt private key", err)
			httputil.HandleError(w, "Could not decrypt private key", http.StatusBadRequest)
			return
		}
		secretKeys = append(secretKeys, secretKey)
	}

	// Make keystore for each secret key
	keystores := make([]*keymanager.Keystore, len(secretKeys))
	passwords := make([]string, len(secretKeys))
	encryptor := keystorev4.New()
	for i := 0; i < len(secretKeys); i++ {
		pubkey := secretKeys[i].PublicKey()
		cryptoFields, err := encryptor.Encrypt(secretKeys[i].Marshal(), tempPassword)
		if err != nil {
			log.WithError(err).Error("Could not encrypt secret key when wrapping it in a keystore")
			httputil.HandleError(w, "Could not encrypt secret key when wrapping it in a keystore", http.StatusInternalServerError)
			return
		}
		k := &keymanager.Keystore{
			Crypto:      cryptoFields,
			ID:          fmt.Sprint(i), // temporary keystore ID
			Pubkey:      fmt.Sprintf("%x", pubkey.Marshal()),
			Version:     encryptor.Version(),
			Description: encryptor.Name(),
		}
		keystores[i] = k
		passwords[i] = tempPassword
	}

	statuses, err := importer.ImportKeystores(ctx, keystores, passwords)
	if err != nil {
		log.WithError(err).Error("Could not import keys", err)
		httputil.HandleError(w, "Could not import keys", http.StatusInternalServerError)
		return
	}

	// Map the statuses to the response
	importedStatuses := make([]*KeystoreStatusData, len(statuses))
	for i, status := range statuses {
		importedStatuses[i] = &KeystoreStatusData{
			PublicKey: "0x" + keystores[i].Pubkey,
			Status:    status,
		}
	}

	httputil.WriteJson(w, &ImportAccountsWithPrivateKeyResponse{
		Data: importedStatuses,
	})
}

func createDepositData(
	ctx context.Context,
	depositPubkey bls.PublicKey,
	eth1WithdrawalAddress []byte,
	amountInGwei uint64,
	signer iface.SigningFunc,
) (*DepositDataResponse, error) {
	depositMessage := &ethpb.DepositMessage{
		PublicKey:             depositPubkey.Marshal(),
		WithdrawalCredentials: eth1WithdrawalCredential(eth1WithdrawalAddress),
		Amount:                amountInGwei,
	}

	sr, err := depositMessage.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	domain, err := signing.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		nil, /*forkVersion*/
		nil, /*genesisValidatorsRoot*/
	)
	if err != nil {
		return nil, err
	}
	root, err := (&ethpb.SigningData{ObjectRoot: sr[:], Domain: domain}).HashTreeRoot()
	if err != nil {
		return nil, err
	}

	sig, err := signer(ctx, &validatorpb.SignRequest{
		PublicKey:   depositPubkey.Marshal(),
		SigningRoot: root[:],
	})
	if err != nil {
		return nil, err
	}

	di := &ethpb.Deposit_Data{
		PublicKey:             depositMessage.PublicKey,
		WithdrawalCredentials: depositMessage.WithdrawalCredentials,
		Amount:                depositMessage.Amount,
		Signature:             sig.Marshal(),
	}

	dr, err := di.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	dd := &DepositDataResponse{
		Pubkey:                depositMessage.PublicKey,
		WithdrawalCredentials: depositMessage.WithdrawalCredentials,
		Signature:             sig.Marshal(),
		DepositDataRoot:       dr[:],
	}
	return dd, nil
}

// eth1WithdrawalCredential wraps eth1 address(20 bytes) into
// eth1 withdrawal credential(32 bytes).
func eth1WithdrawalCredential(eth1WithdrawalAddress []byte) []byte {
	prefix := params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	return append(append([]byte{prefix}, params.BeaconConfig().ZeroHash[1:12]...), eth1WithdrawalAddress[:20]...)[:32]
}

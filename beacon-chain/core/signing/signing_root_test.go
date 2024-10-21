package signing_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestSigningRoot_ComputeSigningRoot(t *testing.T) {
	emptyBlock := util.NewBeaconBlock()
	_, err := signing.ComputeSigningRoot(emptyBlock, bytesutil.PadTo([]byte{'T', 'E', 'S', 'T'}, 32))
	assert.NoError(t, err, "Could not compute signing root of block")
}

func TestSigningRoot_ComputeDomain(t *testing.T) {
	tests := []struct {
		epoch      uint64
		domainType [4]byte
		domain     []byte
	}{
		{epoch: 1, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 208, 77, 177, 79, 38, 245, 35, 111, 26, 208, 21, 163, 231, 235, 15, 122, 232, 16, 119, 107, 178, 238, 117, 43, 187, 117, 240, 228}},
		{epoch: 2, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 208, 77, 177, 79, 38, 245, 35, 111, 26, 208, 21, 163, 231, 235, 15, 122, 232, 16, 119, 107, 178, 238, 117, 43, 187, 117, 240, 228}},
		{epoch: 2, domainType: [4]byte{5, 0, 0, 0}, domain: []byte{5, 0, 0, 0, 208, 77, 177, 79, 38, 245, 35, 111, 26, 208, 21, 163, 231, 235, 15, 122, 232, 16, 119, 107, 178, 238, 117, 43, 187, 117, 240, 228}},
		{epoch: 3, domainType: [4]byte{4, 0, 0, 0}, domain: []byte{4, 0, 0, 0, 208, 77, 177, 79, 38, 245, 35, 111, 26, 208, 21, 163, 231, 235, 15, 122, 232, 16, 119, 107, 178, 238, 117, 43, 187, 117, 240, 228}},
		{epoch: 3, domainType: [4]byte{5, 0, 0, 0}, domain: []byte{5, 0, 0, 0, 208, 77, 177, 79, 38, 245, 35, 111, 26, 208, 21, 163, 231, 235, 15, 122, 232, 16, 119, 107, 178, 238, 117, 43, 187, 117, 240, 228}},
	}
	for _, tt := range tests {
		if got, err := signing.ComputeDomain(tt.domainType, nil, nil); !bytes.Equal(got, tt.domain) {
			t.Errorf("wanted domain version: %d, got: %d", tt.domain, got)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestSigningRoot_ComputeDomainAndSign(t *testing.T) {
	tests := []struct {
		name       string
		genState   func(t *testing.T) (state.BeaconState, []bls.SecretKey)
		genBlock   func(t *testing.T, st state.BeaconState, keys []bls.SecretKey) *ethpb.SignedBeaconBlock
		domainType [4]byte
		want       []byte
	}{
		{
			name: "block proposer",
			genState: func(t *testing.T) (state.BeaconState, []bls.SecretKey) {
				beaconState, privKeys := util.DeterministicGenesisState(t, 100)
				require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
				return beaconState, privKeys
			},
			genBlock: func(t *testing.T, st state.BeaconState, keys []bls.SecretKey) *ethpb.SignedBeaconBlock {
				block, err := util.GenerateFullBlock(st, keys, nil, 1)
				require.NoError(t, err)
				return block
			},
			domainType: params.BeaconConfig().DomainBeaconProposer,
			want: []byte{
				0xb6, 0xa5, 0xea, 0x35, 0x9c, 0x7b, 0x39, 0x78,
				0xda, 0xbb, 0x7b, 0xfa, 0x77, 0x5b, 0xcf, 0x79,
				0x20, 0xaa, 0x2e, 0x01, 0x49, 0x8e, 0xd5, 0x43,
				0x91, 0x75, 0x9a, 0x93, 0x85, 0xa3, 0x4b, 0xa2,
				0x77, 0xdc, 0xf1, 0xc2, 0x00, 0xd5, 0x39, 0x67,
				0xd8, 0x7e, 0xb6, 0x6e, 0x04, 0x9c, 0x30, 0x4b,
				0x0a, 0x40, 0x3d, 0x37, 0xd4, 0x03, 0x02, 0xe7,
				0xe9, 0xa7, 0xae, 0x08, 0x4c, 0x77, 0xde, 0x51,
				0x20, 0x0e, 0x86, 0x14, 0x76, 0x33, 0xb1, 0x27,
				0x98, 0x1e, 0x03, 0x2d, 0x75, 0x44, 0x36, 0xb2,
				0xa6, 0x4c, 0xc7, 0xcd, 0x56, 0x6e, 0x1e, 0xbf,
				0x11, 0xf2, 0xc0, 0x11, 0x90, 0xad, 0xf5, 0x4e,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beaconState, privKeys := tt.genState(t)
			idx, err := helpers.BeaconProposerIndex(context.Background(), beaconState)
			require.NoError(t, err)
			block := tt.genBlock(t, beaconState, privKeys)
			got, err := signing.ComputeDomainAndSign(
				beaconState, time.CurrentEpoch(beaconState), block, tt.domainType, privKeys[idx])
			require.NoError(t, err)
			require.DeepEqual(t, tt.want, got, "Incorrect signature")
		})
	}
}

func TestSigningRoot_ComputeForkDigest(t *testing.T) {
	tests := []struct {
		version []byte
		root    [32]byte
		result  [4]byte
	}{
		{version: []byte{'A', 'B', 'C', 'D'}, root: [32]byte{'i', 'o', 'p'}, result: [4]byte{0x69, 0x5c, 0x26, 0x47}},
		{version: []byte{'i', 'm', 'n', 'a'}, root: [32]byte{'z', 'a', 'b'}, result: [4]byte{0x1c, 0x38, 0x84, 0x58}},
		{version: []byte{'b', 'w', 'r', 't'}, root: [32]byte{'r', 'd', 'c'}, result: [4]byte{0x83, 0x34, 0x38, 0x88}},
	}
	for _, tt := range tests {
		digest, err := signing.ComputeForkDigest(tt.version, tt.root[:])
		require.NoError(t, err)
		assert.Equal(t, tt.result, digest, "Wanted domain version: %#x, got: %#x", digest, tt.result)
	}
}

func TestFuzzverifySigningRoot_10000(_ *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	st := &ethpb.BeaconState{}
	var pubkey [fieldparams.BLSPubkeyLength]byte
	var sig [96]byte
	var domain [4]byte
	var p []byte
	var s []byte
	var d []byte
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(st)
		fuzzer.Fuzz(&pubkey)
		fuzzer.Fuzz(&sig)
		fuzzer.Fuzz(&domain)
		fuzzer.Fuzz(st)
		fuzzer.Fuzz(&p)
		fuzzer.Fuzz(&s)
		fuzzer.Fuzz(&d)
		err := signing.VerifySigningRoot(st, pubkey[:], sig[:], domain[:])
		_ = err
		err = signing.VerifySigningRoot(st, p, s, d)
		_ = err
	}
}

func TestDigestMap(t *testing.T) {
	testVersion := []byte{'A', 'B', 'C', 'D'}
	testValRoot := [32]byte{'t', 'e', 's', 't', 'r', 'o', 'o', 't'}
	digest, err := signing.ComputeForkDigest(testVersion, testValRoot[:])
	assert.NoError(t, err)

	cachedDigest, err := signing.ComputeForkDigest(testVersion, testValRoot[:])
	assert.NoError(t, err)
	assert.Equal(t, digest, cachedDigest)
	testVersion[3] = 'E'
	cachedDigest, err = signing.ComputeForkDigest(testVersion, testValRoot[:])
	assert.NoError(t, err)
	assert.NotEqual(t, digest, cachedDigest)
	testValRoot[5] = 'z'
	cachedDigest2, err := signing.ComputeForkDigest(testVersion, testValRoot[:])
	assert.NoError(t, err)
	assert.NotEqual(t, digest, cachedDigest2)
	assert.NotEqual(t, cachedDigest, cachedDigest2)
}
func TestBlockSignatureBatch_NoSigVerification(t *testing.T) {
	tests := []struct {
		pubkey          []byte
		mockSignature   []byte
		domain          []byte
		wantMessageHexs []string
	}{
		{
			pubkey:          []byte{0xa9, 0x9a, 0x76, 0xed, 0x77, 0x96, 0xf7, 0xbe, 0x22, 0xd5, 0xb7, 0xe8, 0x5d, 0xee, 0xb7, 0xc5, 0x67, 0x7e, 0x88, 0xe5, 0x11, 0xe0, 0xb3, 0x37, 0x61, 0x8f, 0x8c, 0x4e, 0xb6, 0x13, 0x49, 0xb4, 0xbf, 0x2d, 0x15, 0x3f, 0x64, 0x9f, 0x7b, 0x53, 0x35, 0x9f, 0xe8, 0xb9, 0x4a, 0x38, 0xe4, 0x4c},
			mockSignature:   []byte{0xa9, 0x9a, 0x76, 0xed, 0x77},
			domain:          []byte{4, 0, 0, 0, 245, 165, 253, 66, 209, 106, 32, 48, 39, 152, 239, 110, 211, 9, 151, 155, 67, 0, 61, 35, 32, 217, 240, 232, 234, 152, 49, 169},
			wantMessageHexs: []string{"0xe6012bc68e112797a91ed6889e7453f8e304fb76fbffcec1c62eef280a93f7ba"},
		},
	}
	for _, tt := range tests {
		block := util.NewBeaconBlock()
		got, err := signing.BlockSignatureBatch(tt.pubkey, tt.mockSignature, tt.domain, block.Block.HashTreeRoot)
		require.NoError(t, err)
		for i, message := range got.Messages {
			require.Equal(t, hexutil.Encode(message[:]), tt.wantMessageHexs[i])
		}
	}
}

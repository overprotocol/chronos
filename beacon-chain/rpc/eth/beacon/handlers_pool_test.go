package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/api/server"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	blockchainmock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations"
	slashingsmock "github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/slashings/mock"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/voluntaryexits/mock"
	p2pMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v5/network/httputil"
	ethpbv1alpha1 "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
)

func TestListAttestations(t *testing.T) {
	att1 := &ethpbv1alpha1.Attestation{
		AggregationBits: []byte{1, 10},
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            1,
			CommitteeIndex:  1,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature1"), 96),
	}
	att2 := &ethpbv1alpha1.Attestation{
		AggregationBits: []byte{1, 10},
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            1,
			CommitteeIndex:  4,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature2"), 96),
	}
	att3 := &ethpbv1alpha1.Attestation{
		AggregationBits: bitfield.NewBitlist(8),
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            2,
			CommitteeIndex:  2,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot3"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot3"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot3"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature3"), 96),
	}
	att4 := &ethpbv1alpha1.Attestation{
		AggregationBits: bitfield.NewBitlist(8),
		Data: &ethpbv1alpha1.AttestationData{
			Slot:            2,
			CommitteeIndex:  4,
			BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot4"), 32),
			Source: &ethpbv1alpha1.Checkpoint{
				Epoch: 1,
				Root:  bytesutil.PadTo([]byte("sourceroot4"), 32),
			},
			Target: &ethpbv1alpha1.Checkpoint{
				Epoch: 10,
				Root:  bytesutil.PadTo([]byte("targetroot4"), 32),
			},
		},
		Signature: bytesutil.PadTo([]byte("signature4"), 96),
	}
	t.Run("V1", func(t *testing.T) {
		bs, err := util.NewBeaconState()
		require.NoError(t, err)

		chainService := &blockchainmock.ChainService{State: bs}
		s := &Server{
			ChainInfoFetcher: chainService,
			TimeFetcher:      chainService,
			AttestationsPool: attestations.NewPool(),
		}

		require.NoError(t, s.AttestationsPool.SaveAggregatedAttestations([]ethpbv1alpha1.Att{att1, att2}))
		require.NoError(t, s.AttestationsPool.SaveUnaggregatedAttestations([]ethpbv1alpha1.Att{att3, att4}))

		t.Run("empty request", func(t *testing.T) {
			url := "http://example.com"
			request := httptest.NewRequest(http.MethodGet, url, nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.ListAttestations(writer, request)
			assert.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.ListAttestationsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)

			var atts []*structs.Attestation
			require.NoError(t, json.Unmarshal(resp.Data, &atts))
			assert.Equal(t, 4, len(atts))
		})
		t.Run("slot request", func(t *testing.T) {
			url := "http://example.com?slot=2"
			request := httptest.NewRequest(http.MethodGet, url, nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.ListAttestations(writer, request)
			assert.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.ListAttestationsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)

			var atts []*structs.Attestation
			require.NoError(t, json.Unmarshal(resp.Data, &atts))
			assert.Equal(t, 2, len(atts))
			for _, a := range atts {
				assert.Equal(t, "2", a.Data.Slot)
			}
		})
		t.Run("index request", func(t *testing.T) {
			url := "http://example.com?committee_index=4"
			request := httptest.NewRequest(http.MethodGet, url, nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.ListAttestations(writer, request)
			assert.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.ListAttestationsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)

			var atts []*structs.Attestation
			require.NoError(t, json.Unmarshal(resp.Data, &atts))
			assert.Equal(t, 2, len(atts))
			for _, a := range atts {
				assert.Equal(t, "4", a.Data.CommitteeIndex)
			}
		})
		t.Run("both slot + index request", func(t *testing.T) {
			url := "http://example.com?slot=2&committee_index=4"
			request := httptest.NewRequest(http.MethodGet, url, nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.ListAttestations(writer, request)
			assert.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.ListAttestationsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)

			var atts []*structs.Attestation
			require.NoError(t, json.Unmarshal(resp.Data, &atts))
			assert.Equal(t, 1, len(atts))
			for _, a := range atts {
				assert.Equal(t, "2", a.Data.Slot)
				assert.Equal(t, "4", a.Data.CommitteeIndex)
			}
		})
	})
	t.Run("V2", func(t *testing.T) {
		t.Run("Pre-Electra", func(t *testing.T) {
			bs, err := util.NewBeaconState()
			require.NoError(t, err)

			chainService := &blockchainmock.ChainService{State: bs, Genesis: time.Now()}
			s := &Server{
				ChainInfoFetcher: chainService,
				TimeFetcher:      chainService,
				AttestationsPool: attestations.NewPool(),
			}

			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.DenebForkEpoch = 0
			config.AlpacaForkEpoch = 100
			params.OverrideBeaconConfig(config)

			require.NoError(t, s.AttestationsPool.SaveAggregatedAttestations([]ethpbv1alpha1.Att{att1, att2}))
			require.NoError(t, s.AttestationsPool.SaveUnaggregatedAttestations([]ethpbv1alpha1.Att{att3, att4}))
			t.Run("empty request", func(t *testing.T) {
				url := "http://example.com"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.Attestation
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 4, len(atts))
				assert.Equal(t, "deneb", resp.Version)
			})
			t.Run("slot request", func(t *testing.T) {
				url := "http://example.com?slot=2"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.Attestation
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 2, len(atts))
				assert.Equal(t, "deneb", resp.Version)
				for _, a := range atts {
					assert.Equal(t, "2", a.Data.Slot)
				}
			})
			t.Run("index request", func(t *testing.T) {
				url := "http://example.com?committee_index=4"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.Attestation
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 2, len(atts))
				assert.Equal(t, "deneb", resp.Version)
				for _, a := range atts {
					assert.Equal(t, "4", a.Data.CommitteeIndex)
				}
			})
			t.Run("both slot + index request", func(t *testing.T) {
				url := "http://example.com?slot=2&committee_index=4"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.Attestation
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 1, len(atts))
				assert.Equal(t, "deneb", resp.Version)
				for _, a := range atts {
					assert.Equal(t, "2", a.Data.Slot)
					assert.Equal(t, "4", a.Data.CommitteeIndex)
				}
			})
		})
		t.Run("Post-Electra", func(t *testing.T) {
			cb := primitives.NewAttestationCommitteeBits()
			cb.SetBitAt(1, true)
			attElectra1 := &ethpbv1alpha1.AttestationElectra{
				AggregationBits: []byte{1, 10},
				Data: &ethpbv1alpha1.AttestationData{
					Slot:            1,
					CommitteeIndex:  1,
					BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
					Source: &ethpbv1alpha1.Checkpoint{
						Epoch: 1,
						Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
					},
					Target: &ethpbv1alpha1.Checkpoint{
						Epoch: 10,
						Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
					},
				},
				CommitteeBits: cb,
				Signature:     bytesutil.PadTo([]byte("signature1"), 96),
			}
			attElectra2 := &ethpbv1alpha1.AttestationElectra{
				AggregationBits: []byte{1, 10},
				Data: &ethpbv1alpha1.AttestationData{
					Slot:            1,
					CommitteeIndex:  4,
					BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
					Source: &ethpbv1alpha1.Checkpoint{
						Epoch: 1,
						Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
					},
					Target: &ethpbv1alpha1.Checkpoint{
						Epoch: 10,
						Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
					},
				},
				CommitteeBits: cb,
				Signature:     bytesutil.PadTo([]byte("signature2"), 96),
			}
			attElectra3 := &ethpbv1alpha1.AttestationElectra{
				AggregationBits: bitfield.NewBitlist(8),
				Data: &ethpbv1alpha1.AttestationData{
					Slot:            2,
					CommitteeIndex:  2,
					BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot3"), 32),
					Source: &ethpbv1alpha1.Checkpoint{
						Epoch: 1,
						Root:  bytesutil.PadTo([]byte("sourceroot3"), 32),
					},
					Target: &ethpbv1alpha1.Checkpoint{
						Epoch: 10,
						Root:  bytesutil.PadTo([]byte("targetroot3"), 32),
					},
				},
				CommitteeBits: cb,
				Signature:     bytesutil.PadTo([]byte("signature3"), 96),
			}
			attElectra4 := &ethpbv1alpha1.AttestationElectra{
				AggregationBits: bitfield.NewBitlist(8),
				Data: &ethpbv1alpha1.AttestationData{
					Slot:            2,
					CommitteeIndex:  4,
					BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot4"), 32),
					Source: &ethpbv1alpha1.Checkpoint{
						Epoch: 1,
						Root:  bytesutil.PadTo([]byte("sourceroot4"), 32),
					},
					Target: &ethpbv1alpha1.Checkpoint{
						Epoch: 10,
						Root:  bytesutil.PadTo([]byte("targetroot4"), 32),
					},
				},
				CommitteeBits: cb,
				Signature:     bytesutil.PadTo([]byte("signature4"), 96),
			}
			bs, err := util.NewBeaconStateElectra()
			require.NoError(t, err)

			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.AlpacaForkEpoch = 0
			params.OverrideBeaconConfig(config)

			chainService := &blockchainmock.ChainService{State: bs}
			s := &Server{
				AttestationsPool: attestations.NewPool(),
				ChainInfoFetcher: chainService,
				TimeFetcher:      chainService,
			}
			// Added one pre electra attestation to ensure it is ignored.
			require.NoError(t, s.AttestationsPool.SaveAggregatedAttestations([]ethpbv1alpha1.Att{attElectra1, attElectra2, att1}))
			require.NoError(t, s.AttestationsPool.SaveUnaggregatedAttestations([]ethpbv1alpha1.Att{attElectra3, attElectra4, att3}))

			t.Run("empty request", func(t *testing.T) {
				url := "http://example.com"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.AttestationElectra
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 4, len(atts))
				assert.Equal(t, "alpaca", resp.Version)
			})
			t.Run("slot request", func(t *testing.T) {
				url := "http://example.com?slot=2"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.AttestationElectra
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 2, len(atts))
				assert.Equal(t, "alpaca", resp.Version)
				for _, a := range atts {
					assert.Equal(t, "2", a.Data.Slot)
				}
			})
			t.Run("index request", func(t *testing.T) {
				url := "http://example.com?committee_index=4"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.AttestationElectra
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 2, len(atts))
				assert.Equal(t, "alpaca", resp.Version)
				for _, a := range atts {
					assert.Equal(t, "4", a.Data.CommitteeIndex)
				}
			})
			t.Run("both slot + index request", func(t *testing.T) {
				url := "http://example.com?slot=2&committee_index=4"
				request := httptest.NewRequest(http.MethodGet, url, nil)
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.ListAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				resp := &structs.ListAttestationsResponse{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
				require.NotNil(t, resp)
				require.NotNil(t, resp.Data)

				var atts []*structs.AttestationElectra
				require.NoError(t, json.Unmarshal(resp.Data, &atts))
				assert.Equal(t, 1, len(atts))
				assert.Equal(t, "alpaca", resp.Version)
				for _, a := range atts {
					assert.Equal(t, "2", a.Data.Slot)
					assert.Equal(t, "4", a.Data.CommitteeIndex)
				}
			})
		})
	})
}

func TestSubmitAttestations(t *testing.T) {
	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	params.SetupTestConfigCleanup(t)
	c := params.BeaconConfig().Copy()
	// Required for correct committee size calculation.
	c.SlotsPerEpoch = 1
	params.OverrideBeaconConfig(c)

	_, keys, err := util.DeterministicDepositsAndKeys(1)
	require.NoError(t, err)
	validators := []*ethpbv1alpha1.Validator{
		{
			PublicKey: keys[0].PublicKey().Marshal(),
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
		state.Validators = validators
		state.Slot = 1
		state.PreviousJustifiedCheckpoint = &ethpbv1alpha1.Checkpoint{
			Epoch: 0,
			Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
		}
		return nil
	})
	require.NoError(t, err)
	b := bitfield.NewBitlist(1)
	b.SetBitAt(0, true)

	chainService := &blockchainmock.ChainService{State: bs}
	s := &Server{
		HeadFetcher:       chainService,
		ChainInfoFetcher:  chainService,
		OperationNotifier: &blockchainmock.MockOperationNotifier{},
	}
	t.Run("V1", func(t *testing.T) {
		t.Run("single", func(t *testing.T) {
			broadcaster := &p2pMock.MockBroadcaster{}
			s.Broadcaster = broadcaster
			s.AttestationsPool = attestations.NewPool()

			var body bytes.Buffer
			_, err := body.WriteString(singleAtt)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttestations(writer, request)

			assert.Equal(t, http.StatusOK, writer.Code)
			assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
			assert.Equal(t, 1, broadcaster.NumAttestations())
			assert.Equal(t, "0x03", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetAggregationBits()))
			assert.Equal(t, "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetSignature()))
			assert.Equal(t, primitives.Slot(0), broadcaster.BroadcastAttestations[0].GetData().Slot)
			assert.Equal(t, primitives.CommitteeIndex(0), broadcaster.BroadcastAttestations[0].GetData().CommitteeIndex)
			assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().BeaconBlockRoot))
			assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().Source.Root))
			assert.Equal(t, primitives.Epoch(0), broadcaster.BroadcastAttestations[0].GetData().Source.Epoch)
			assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().Target.Root))
			assert.Equal(t, primitives.Epoch(0), broadcaster.BroadcastAttestations[0].GetData().Target.Epoch)
			assert.Equal(t, 1, s.AttestationsPool.UnaggregatedAttestationCount())
		})
		t.Run("multiple", func(t *testing.T) {
			broadcaster := &p2pMock.MockBroadcaster{}
			s.Broadcaster = broadcaster
			s.AttestationsPool = attestations.NewPool()

			var body bytes.Buffer
			_, err := body.WriteString(multipleAtts)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttestations(writer, request)
			assert.Equal(t, http.StatusOK, writer.Code)
			assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
			assert.Equal(t, 2, broadcaster.NumAttestations())
			assert.Equal(t, 2, s.AttestationsPool.UnaggregatedAttestationCount())
		})
		t.Run("no body", func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttestations(writer, request)
			assert.Equal(t, http.StatusBadRequest, writer.Code)
			e := &httputil.DefaultJsonError{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
			assert.Equal(t, http.StatusBadRequest, e.Code)
			assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
		})
		t.Run("empty", func(t *testing.T) {
			var body bytes.Buffer
			_, err := body.WriteString("[]")
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttestations(writer, request)
			assert.Equal(t, http.StatusBadRequest, writer.Code)
			e := &httputil.DefaultJsonError{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
			assert.Equal(t, http.StatusBadRequest, e.Code)
			assert.Equal(t, true, strings.Contains(e.Message, "no data submitted"))
		})
		t.Run("invalid", func(t *testing.T) {
			var body bytes.Buffer
			_, err := body.WriteString(invalidAtt)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttestations(writer, request)
			assert.Equal(t, http.StatusBadRequest, writer.Code)
			e := &server.IndexedVerificationFailureError{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
			assert.Equal(t, http.StatusBadRequest, e.Code)
			require.Equal(t, 1, len(e.Failures))
			assert.Equal(t, true, strings.Contains(e.Failures[0].Message, "Incorrect attestation signature"))
		})
	})
	t.Run("V2", func(t *testing.T) {
		t.Run("pre-electra", func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				broadcaster := &p2pMock.MockBroadcaster{}
				s.Broadcaster = broadcaster
				s.AttestationsPool = attestations.NewPool()

				var body bytes.Buffer
				_, err := body.WriteString(singleAtt)
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Phase0))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)

				assert.Equal(t, http.StatusOK, writer.Code)
				assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
				assert.Equal(t, 1, broadcaster.NumAttestations())
				assert.Equal(t, "0x03", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetAggregationBits()))
				assert.Equal(t, "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetSignature()))
				assert.Equal(t, primitives.Slot(0), broadcaster.BroadcastAttestations[0].GetData().Slot)
				assert.Equal(t, primitives.CommitteeIndex(0), broadcaster.BroadcastAttestations[0].GetData().CommitteeIndex)
				assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().BeaconBlockRoot))
				assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().Source.Root))
				assert.Equal(t, primitives.Epoch(0), broadcaster.BroadcastAttestations[0].GetData().Source.Epoch)
				assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().Target.Root))
				assert.Equal(t, primitives.Epoch(0), broadcaster.BroadcastAttestations[0].GetData().Target.Epoch)
				assert.Equal(t, 1, s.AttestationsPool.UnaggregatedAttestationCount())
			})
			t.Run("multiple", func(t *testing.T) {
				broadcaster := &p2pMock.MockBroadcaster{}
				s.Broadcaster = broadcaster
				s.AttestationsPool = attestations.NewPool()

				var body bytes.Buffer
				_, err := body.WriteString(multipleAtts)
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Phase0))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
				assert.Equal(t, 2, broadcaster.NumAttestations())
				assert.Equal(t, 2, s.AttestationsPool.UnaggregatedAttestationCount())
			})
			t.Run("no body", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
				request.Header.Set(api.VersionHeader, version.String(version.Phase0))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusBadRequest, writer.Code)
				e := &httputil.DefaultJsonError{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
				assert.Equal(t, http.StatusBadRequest, e.Code)
				assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
			})
			t.Run("empty", func(t *testing.T) {
				var body bytes.Buffer
				_, err := body.WriteString("[]")
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Phase0))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusBadRequest, writer.Code)
				e := &httputil.DefaultJsonError{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
				assert.Equal(t, http.StatusBadRequest, e.Code)
				assert.Equal(t, true, strings.Contains(e.Message, "no data submitted"))
			})
			t.Run("invalid", func(t *testing.T) {
				var body bytes.Buffer
				_, err := body.WriteString(invalidAtt)
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Phase0))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusBadRequest, writer.Code)
				e := &server.IndexedVerificationFailureError{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
				assert.Equal(t, http.StatusBadRequest, e.Code)
				require.Equal(t, 1, len(e.Failures))
				assert.Equal(t, true, strings.Contains(e.Failures[0].Message, "Incorrect attestation signature"))
			})
		})
		t.Run("post-electra", func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				broadcaster := &p2pMock.MockBroadcaster{}
				s.Broadcaster = broadcaster
				s.AttestationsPool = attestations.NewPool()

				var body bytes.Buffer
				_, err := body.WriteString(singleAttElectra)
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)

				assert.Equal(t, http.StatusOK, writer.Code)
				assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
				assert.Equal(t, 1, broadcaster.NumAttestations())
				assert.Equal(t, "0x03", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetAggregationBits()))
				assert.Equal(t, "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetSignature()))
				assert.Equal(t, primitives.Slot(0), broadcaster.BroadcastAttestations[0].GetData().Slot)
				assert.Equal(t, primitives.CommitteeIndex(0), broadcaster.BroadcastAttestations[0].GetData().CommitteeIndex)
				assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().BeaconBlockRoot))
				assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().Source.Root))
				assert.Equal(t, primitives.Epoch(0), broadcaster.BroadcastAttestations[0].GetData().Source.Epoch)
				assert.Equal(t, "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2", hexutil.Encode(broadcaster.BroadcastAttestations[0].GetData().Target.Root))
				assert.Equal(t, primitives.Epoch(0), broadcaster.BroadcastAttestations[0].GetData().Target.Epoch)
				assert.Equal(t, 1, s.AttestationsPool.UnaggregatedAttestationCount())
			})
			t.Run("multiple", func(t *testing.T) {
				broadcaster := &p2pMock.MockBroadcaster{}
				s.Broadcaster = broadcaster
				s.AttestationsPool = attestations.NewPool()

				var body bytes.Buffer
				_, err := body.WriteString(multipleAttsElectra)
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusOK, writer.Code)
				assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
				assert.Equal(t, 2, broadcaster.NumAttestations())
				assert.Equal(t, 2, s.AttestationsPool.UnaggregatedAttestationCount())
			})
			t.Run("no body", func(t *testing.T) {
				request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
				request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusBadRequest, writer.Code)
				e := &httputil.DefaultJsonError{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
				assert.Equal(t, http.StatusBadRequest, e.Code)
				assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
			})
			t.Run("empty", func(t *testing.T) {
				var body bytes.Buffer
				_, err := body.WriteString("[]")
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusBadRequest, writer.Code)
				e := &httputil.DefaultJsonError{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
				assert.Equal(t, http.StatusBadRequest, e.Code)
				assert.Equal(t, true, strings.Contains(e.Message, "no data submitted"))
			})
			t.Run("invalid", func(t *testing.T) {
				var body bytes.Buffer
				_, err := body.WriteString(invalidAttElectra)
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
				request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
				writer := httptest.NewRecorder()
				writer.Body = &bytes.Buffer{}

				s.SubmitAttestationsV2(writer, request)
				assert.Equal(t, http.StatusBadRequest, writer.Code)
				e := &server.IndexedVerificationFailureError{}
				require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
				assert.Equal(t, http.StatusBadRequest, e.Code)
				require.Equal(t, 1, len(e.Failures))
				assert.Equal(t, true, strings.Contains(e.Failures[0].Message, "Incorrect attestation signature"))
			})
		})
	})

}

func TestListVoluntaryExits(t *testing.T) {
	exit1 := &ethpbv1alpha1.SignedVoluntaryExit{
		Exit: &ethpbv1alpha1.VoluntaryExit{
			Epoch:          1,
			ValidatorIndex: 1,
		},
		Signature: bytesutil.PadTo([]byte("signature1"), 96),
	}
	exit2 := &ethpbv1alpha1.SignedVoluntaryExit{
		Exit: &ethpbv1alpha1.VoluntaryExit{
			Epoch:          2,
			ValidatorIndex: 2,
		},
		Signature: bytesutil.PadTo([]byte("signature2"), 96),
	}

	s := &Server{
		VoluntaryExitsPool: &mock.PoolMock{Exits: []*ethpbv1alpha1.SignedVoluntaryExit{exit1, exit2}},
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.ListVoluntaryExits(writer, request)
	assert.Equal(t, http.StatusOK, writer.Code)
	resp := &structs.ListVoluntaryExitsResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.Data)
	require.Equal(t, 2, len(resp.Data))
	assert.Equal(t, "0x7369676e6174757265310000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", resp.Data[0].Signature)
	assert.Equal(t, "1", resp.Data[0].Message.Epoch)
	assert.Equal(t, "1", resp.Data[0].Message.ValidatorIndex)
	assert.Equal(t, "0x7369676e6174757265320000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", resp.Data[1].Signature)
	assert.Equal(t, "2", resp.Data[1].Message.Epoch)
	assert.Equal(t, "2", resp.Data[1].Message.ValidatorIndex)
}

func TestSubmitVoluntaryExit(t *testing.T) {
	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	t.Run("ok", func(t *testing.T) {
		_, keys, err := util.DeterministicDepositsAndKeys(1)
		require.NoError(t, err)
		validator := &ethpbv1alpha1.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[0].PublicKey().Marshal(),
		}
		bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
			state.Validators = []*ethpbv1alpha1.Validator{validator}
			// Satisfy activity time required before exiting.
			state.Slot = params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod))
			return nil
		})
		require.NoError(t, err)

		broadcaster := &p2pMock.MockBroadcaster{}
		s := &Server{
			ChainInfoFetcher:   &blockchainmock.ChainService{State: bs},
			VoluntaryExitsPool: &mock.PoolMock{},
			Broadcaster:        broadcaster,
		}

		var body bytes.Buffer
		_, err = body.WriteString(exit1)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitVoluntaryExit(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		require.NoError(t, err)
		pendingExits, err := s.VoluntaryExitsPool.PendingExits()
		require.NoError(t, err)
		require.Equal(t, 1, len(pendingExits))
		assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
	})
	t.Run("across fork", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		config := params.BeaconConfig()
		config.AltairForkEpoch = params.BeaconConfig().ShardCommitteePeriod + 1
		config.BellatrixForkEpoch = params.BeaconConfig().ShardCommitteePeriod + 32
		params.OverrideBeaconConfig(config)

		bs, _ := util.DeterministicGenesisStateAltair(t, 1)
		require.NoError(t, bs.SetSlot(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().ShardCommitteePeriod+31))))
		broadcaster := &p2pMock.MockBroadcaster{}
		s := &Server{
			ChainInfoFetcher:   &blockchainmock.ChainService{State: bs},
			VoluntaryExitsPool: &mock.PoolMock{},
			Broadcaster:        broadcaster,
		}

		var body bytes.Buffer
		_, err := body.WriteString(exit2)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitVoluntaryExit(writer, request)
		assert.Equal(t, http.StatusOK, writer.Code)
		require.NoError(t, err)
		pendingExits, err := s.VoluntaryExitsPool.PendingExits()
		require.NoError(t, err)
		require.Equal(t, 1, len(pendingExits))
		assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
	})
	t.Run("no body", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s := &Server{}
		s.SubmitVoluntaryExit(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "No data submitted"))
	})
	t.Run("invalid", func(t *testing.T) {
		var body bytes.Buffer
		_, err := body.WriteString(invalidExit1)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s := &Server{}
		s.SubmitVoluntaryExit(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
	})
	t.Run("wrong signature", func(t *testing.T) {
		bs, _ := util.DeterministicGenesisState(t, 1)
		s := &Server{ChainInfoFetcher: &blockchainmock.ChainService{State: bs}}

		var body bytes.Buffer
		_, err := body.WriteString(invalidExit2)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitVoluntaryExit(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Invalid exit"))
	})
	t.Run("invalid validator index", func(t *testing.T) {
		_, keys, err := util.DeterministicDepositsAndKeys(1)
		require.NoError(t, err)
		validator := &ethpbv1alpha1.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[0].PublicKey().Marshal(),
		}
		bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
			state.Validators = []*ethpbv1alpha1.Validator{validator}
			return nil
		})
		require.NoError(t, err)

		s := &Server{ChainInfoFetcher: &blockchainmock.ChainService{State: bs}}

		var body bytes.Buffer
		_, err = body.WriteString(invalidExit3)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com", &body)
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitVoluntaryExit(writer, request)
		assert.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.Equal(t, true, strings.Contains(e.Message, "Could not get validator"))
	})
}

func TestGetAttesterSlashings(t *testing.T) {
	slashing1PreElectra := &ethpbv1alpha1.AttesterSlashing{
		Attestation_1: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{1, 10},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            1,
				CommitteeIndex:  1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature1"), 96),
		},
		Attestation_2: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{2, 20},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            2,
				CommitteeIndex:  2,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 2,
					Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 20,
					Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature2"), 96),
		},
	}
	slashing2PreElectra := &ethpbv1alpha1.AttesterSlashing{
		Attestation_1: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{3, 30},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            3,
				CommitteeIndex:  3,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot3"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 3,
					Root:  bytesutil.PadTo([]byte("sourceroot3"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 30,
					Root:  bytesutil.PadTo([]byte("targetroot3"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature3"), 96),
		},
		Attestation_2: &ethpbv1alpha1.IndexedAttestation{
			AttestingIndices: []uint64{4, 40},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            4,
				CommitteeIndex:  4,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot4"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 4,
					Root:  bytesutil.PadTo([]byte("sourceroot4"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 40,
					Root:  bytesutil.PadTo([]byte("targetroot4"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature4"), 96),
		},
	}
	slashing1PostElectra := &ethpbv1alpha1.AttesterSlashingElectra{
		Attestation_1: &ethpbv1alpha1.IndexedAttestationElectra{
			AttestingIndices: []uint64{1, 10},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            1,
				CommitteeIndex:  1,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 1,
					Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 10,
					Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature1"), 96),
		},
		Attestation_2: &ethpbv1alpha1.IndexedAttestationElectra{
			AttestingIndices: []uint64{2, 20},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            2,
				CommitteeIndex:  2,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 2,
					Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 20,
					Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature2"), 96),
		},
	}
	slashing2PostElectra := &ethpbv1alpha1.AttesterSlashingElectra{
		Attestation_1: &ethpbv1alpha1.IndexedAttestationElectra{
			AttestingIndices: []uint64{3, 30},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            3,
				CommitteeIndex:  3,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot3"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 3,
					Root:  bytesutil.PadTo([]byte("sourceroot3"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 30,
					Root:  bytesutil.PadTo([]byte("targetroot3"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature3"), 96),
		},
		Attestation_2: &ethpbv1alpha1.IndexedAttestationElectra{
			AttestingIndices: []uint64{4, 40},
			Data: &ethpbv1alpha1.AttestationData{
				Slot:            4,
				CommitteeIndex:  4,
				BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot4"), 32),
				Source: &ethpbv1alpha1.Checkpoint{
					Epoch: 4,
					Root:  bytesutil.PadTo([]byte("sourceroot4"), 32),
				},
				Target: &ethpbv1alpha1.Checkpoint{
					Epoch: 40,
					Root:  bytesutil.PadTo([]byte("targetroot4"), 32),
				},
			},
			Signature: bytesutil.PadTo([]byte("signature4"), 96),
		},
	}

	t.Run("V1", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			bs, err := util.NewBeaconState()
			require.NoError(t, err)

			s := &Server{
				ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
				SlashingsPool:    &slashingsmock.PoolMock{PendingAttSlashings: []ethpbv1alpha1.AttSlashing{slashing1PreElectra, slashing2PreElectra}},
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v1/beacon/pool/attester_slashings", nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.GetAttesterSlashings(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.GetAttesterSlashingsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)

			var slashings []*structs.AttesterSlashing
			require.NoError(t, json.Unmarshal(resp.Data, &slashings))

			ss, err := structs.AttesterSlashingsToConsensus(slashings)
			require.NoError(t, err)

			require.DeepEqual(t, slashing1PreElectra, ss[0])
			require.DeepEqual(t, slashing2PreElectra, ss[1])
		})
		t.Run("no slashings", func(t *testing.T) {
			bs, err := util.NewBeaconState()
			require.NoError(t, err)

			s := &Server{
				ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
				SlashingsPool:    &slashingsmock.PoolMock{PendingAttSlashings: []ethpbv1alpha1.AttSlashing{}},
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v1/beacon/pool/attester_slashings", nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.GetAttesterSlashings(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.GetAttesterSlashingsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)

			var slashings []*structs.AttesterSlashing
			require.NoError(t, json.Unmarshal(resp.Data, &slashings))
			require.Equal(t, 0, len(slashings))
		})
	})
	t.Run("V2", func(t *testing.T) {
		t.Run("post-alpaca-ok-1-pre-slashing", func(t *testing.T) {
			bs, err := util.NewBeaconStateElectra()
			require.NoError(t, err)

			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.AlpacaForkEpoch = 100
			params.OverrideBeaconConfig(config)

			chainService := &blockchainmock.ChainService{State: bs}

			s := &Server{
				ChainInfoFetcher: chainService,
				TimeFetcher:      chainService,
				SlashingsPool:    &slashingsmock.PoolMock{PendingAttSlashings: []ethpbv1alpha1.AttSlashing{slashing1PostElectra, slashing2PostElectra, slashing1PreElectra}},
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v2/beacon/pool/attester_slashings", nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.GetAttesterSlashingsV2(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.GetAttesterSlashingsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)
			assert.Equal(t, "alpaca", resp.Version)

			// Unmarshal resp.Data into a slice of slashings
			var slashings []*structs.AttesterSlashingElectra
			require.NoError(t, json.Unmarshal(resp.Data, &slashings))

			ss, err := structs.AttesterSlashingsElectraToConsensus(slashings)
			require.NoError(t, err)

			require.DeepEqual(t, slashing1PostElectra, ss[0])
			require.DeepEqual(t, slashing2PostElectra, ss[1])
		})
		t.Run("post-alpaca-ok", func(t *testing.T) {
			bs, err := util.NewBeaconStateElectra()
			require.NoError(t, err)

			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.AlpacaForkEpoch = 100
			params.OverrideBeaconConfig(config)

			chainService := &blockchainmock.ChainService{State: bs}

			s := &Server{
				ChainInfoFetcher: chainService,
				TimeFetcher:      chainService,
				SlashingsPool:    &slashingsmock.PoolMock{PendingAttSlashings: []ethpbv1alpha1.AttSlashing{slashing1PostElectra, slashing2PostElectra}},
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v2/beacon/pool/attester_slashings", nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.GetAttesterSlashingsV2(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.GetAttesterSlashingsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)
			assert.Equal(t, "alpaca", resp.Version)

			// Unmarshal resp.Data into a slice of slashings
			var slashings []*structs.AttesterSlashingElectra
			require.NoError(t, json.Unmarshal(resp.Data, &slashings))

			ss, err := structs.AttesterSlashingsElectraToConsensus(slashings)
			require.NoError(t, err)

			require.DeepEqual(t, slashing1PostElectra, ss[0])
			require.DeepEqual(t, slashing2PostElectra, ss[1])
		})
		t.Run("pre-alpaca-ok", func(t *testing.T) {
			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.DenebForkEpoch = 0
			config.AlpacaForkEpoch = 100
			params.OverrideBeaconConfig(config)

			bs, err := util.NewBeaconState()
			require.NoError(t, err)
			chainService := &blockchainmock.ChainService{State: bs, Genesis: time.Now()}

			s := &Server{
				ChainInfoFetcher: chainService,
				TimeFetcher:      chainService,
				SlashingsPool:    &slashingsmock.PoolMock{PendingAttSlashings: []ethpbv1alpha1.AttSlashing{slashing1PreElectra, slashing2PreElectra}},
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v1/beacon/pool/attester_slashings", nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.GetAttesterSlashingsV2(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.GetAttesterSlashingsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)

			var slashings []*structs.AttesterSlashing
			require.NoError(t, json.Unmarshal(resp.Data, &slashings))

			ss, err := structs.AttesterSlashingsToConsensus(slashings)
			require.NoError(t, err)

			require.DeepEqual(t, slashing1PreElectra, ss[0])
			require.DeepEqual(t, slashing2PreElectra, ss[1])
		})
		t.Run("no-slashings", func(t *testing.T) {
			bs, err := util.NewBeaconStateElectra()
			require.NoError(t, err)

			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.AlpacaForkEpoch = 100
			params.OverrideBeaconConfig(config)

			chainService := &blockchainmock.ChainService{State: bs}
			s := &Server{
				ChainInfoFetcher: chainService,
				TimeFetcher:      chainService,
				SlashingsPool:    &slashingsmock.PoolMock{PendingAttSlashings: []ethpbv1alpha1.AttSlashing{}},
			}

			request := httptest.NewRequest(http.MethodGet, "http://example.com/eth/v2/beacon/pool/attester_slashings", nil)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.GetAttesterSlashingsV2(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			resp := &structs.GetAttesterSlashingsResponse{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
			require.NotNil(t, resp)
			require.NotNil(t, resp.Data)
			assert.Equal(t, "alpaca", resp.Version)

			// Unmarshal resp.Data into a slice of slashings
			var slashings []*structs.AttesterSlashingElectra
			require.NoError(t, json.Unmarshal(resp.Data, &slashings))
			require.Equal(t, 0, len(slashings))
		})
	})
}

func TestGetProposerSlashings(t *testing.T) {
	bs, err := util.NewBeaconState()
	require.NoError(t, err)
	slashing1 := &ethpbv1alpha1.ProposerSlashing{
		Header_1: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          1,
				ProposerIndex: 1,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot1"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot1"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot1"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature1"), 96),
		},
		Header_2: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          2,
				ProposerIndex: 2,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot2"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot2"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot2"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature2"), 96),
		},
	}
	slashing2 := &ethpbv1alpha1.ProposerSlashing{
		Header_1: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          3,
				ProposerIndex: 3,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot3"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot3"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot3"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature3"), 96),
		},
		Header_2: &ethpbv1alpha1.SignedBeaconBlockHeader{
			Header: &ethpbv1alpha1.BeaconBlockHeader{
				Slot:          4,
				ProposerIndex: 4,
				ParentRoot:    bytesutil.PadTo([]byte("parentroot4"), 32),
				StateRoot:     bytesutil.PadTo([]byte("stateroot4"), 32),
				BodyRoot:      bytesutil.PadTo([]byte("bodyroot4"), 32),
			},
			Signature: bytesutil.PadTo([]byte("signature4"), 96),
		},
	}

	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{PendingPropSlashings: []*ethpbv1alpha1.ProposerSlashing{slashing1, slashing2}},
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/beacon/pool/attester_slashings", nil)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.GetProposerSlashings(writer, request)
	require.Equal(t, http.StatusOK, writer.Code)
	resp := &structs.GetProposerSlashingsResponse{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.Data)
	assert.Equal(t, 2, len(resp.Data))
}

func TestSubmitAttesterSlashings(t *testing.T) {
	ctx := context.Background()

	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	attestationData1 := &ethpbv1alpha1.AttestationData{
		CommitteeIndex:  1,
		BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot1"), 32),
		Source: &ethpbv1alpha1.Checkpoint{
			Epoch: 1,
			Root:  bytesutil.PadTo([]byte("sourceroot1"), 32),
		},
		Target: &ethpbv1alpha1.Checkpoint{
			Epoch: 10,
			Root:  bytesutil.PadTo([]byte("targetroot1"), 32),
		},
	}
	attestationData2 := &ethpbv1alpha1.AttestationData{
		CommitteeIndex:  1,
		BeaconBlockRoot: bytesutil.PadTo([]byte("blockroot2"), 32),
		Source: &ethpbv1alpha1.Checkpoint{
			Epoch: 1,
			Root:  bytesutil.PadTo([]byte("sourceroot2"), 32),
		},
		Target: &ethpbv1alpha1.Checkpoint{
			Epoch: 10,
			Root:  bytesutil.PadTo([]byte("targetroot2"), 32),
		},
	}

	t.Run("V1", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			attestationData1.Slot = 1
			attestationData2.Slot = 1
			slashing := &ethpbv1alpha1.AttesterSlashing{
				Attestation_1: &ethpbv1alpha1.IndexedAttestation{
					AttestingIndices: []uint64{0},
					Data:             attestationData1,
					Signature:        make([]byte, 96),
				},
				Attestation_2: &ethpbv1alpha1.IndexedAttestation{
					AttestingIndices: []uint64{0},
					Data:             attestationData2,
					Signature:        make([]byte, 96),
				},
			}

			_, keys, err := util.DeterministicDepositsAndKeys(1)
			require.NoError(t, err)
			validator := &ethpbv1alpha1.Validator{
				PublicKey: keys[0].PublicKey().Marshal(),
			}

			bs, err := util.NewBeaconState(func(state *ethpbv1alpha1.BeaconState) error {
				state.Validators = []*ethpbv1alpha1.Validator{validator}
				return nil
			})
			require.NoError(t, err)

			for _, att := range []*ethpbv1alpha1.IndexedAttestation{slashing.Attestation_1, slashing.Attestation_2} {
				sb, err := signing.ComputeDomainAndSign(bs, att.Data.Target.Epoch, att.Data, params.BeaconConfig().DomainBeaconAttester, keys[0])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				att.Signature = sig.Marshal()
			}

			chainmock := &blockchainmock.ChainService{State: bs}
			broadcaster := &p2pMock.MockBroadcaster{}
			s := &Server{
				ChainInfoFetcher:  chainmock,
				SlashingsPool:     &slashingsmock.PoolMock{},
				Broadcaster:       broadcaster,
				OperationNotifier: chainmock.OperationNotifier(),
			}

			toSubmit := structs.AttesterSlashingsFromConsensus([]*ethpbv1alpha1.AttesterSlashing{slashing})
			b, err := json.Marshal(toSubmit[0])
			require.NoError(t, err)
			var body bytes.Buffer
			_, err = body.Write(b)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com/beacon/pool/attester_slashings", &body)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttesterSlashings(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			pendingSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, bs, true)
			require.Equal(t, 1, len(pendingSlashings))
			assert.DeepEqual(t, slashing, pendingSlashings[0])
			require.Equal(t, 1, broadcaster.NumMessages())
			assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
			_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.AttesterSlashing)
			assert.Equal(t, true, ok)
		})
		t.Run("accross-fork", func(t *testing.T) {
			attestationData1.Slot = params.BeaconConfig().SlotsPerEpoch
			attestationData2.Slot = params.BeaconConfig().SlotsPerEpoch
			slashing := &ethpbv1alpha1.AttesterSlashing{
				Attestation_1: &ethpbv1alpha1.IndexedAttestation{
					AttestingIndices: []uint64{0},
					Data:             attestationData1,
					Signature:        make([]byte, 96),
				},
				Attestation_2: &ethpbv1alpha1.IndexedAttestation{
					AttestingIndices: []uint64{0},
					Data:             attestationData2,
					Signature:        make([]byte, 96),
				},
			}

			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.AltairForkEpoch = 1
			params.OverrideBeaconConfig(config)

			bs, keys := util.DeterministicGenesisState(t, 1)
			newBs := bs.Copy()
			newBs, err := transition.ProcessSlots(ctx, newBs, params.BeaconConfig().SlotsPerEpoch)
			require.NoError(t, err)

			for _, att := range []*ethpbv1alpha1.IndexedAttestation{slashing.Attestation_1, slashing.Attestation_2} {
				sb, err := signing.ComputeDomainAndSign(newBs, att.Data.Target.Epoch, att.Data, params.BeaconConfig().DomainBeaconAttester, keys[0])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				att.Signature = sig.Marshal()
			}

			broadcaster := &p2pMock.MockBroadcaster{}
			chainmock := &blockchainmock.ChainService{State: bs}
			s := &Server{
				ChainInfoFetcher:  chainmock,
				SlashingsPool:     &slashingsmock.PoolMock{},
				Broadcaster:       broadcaster,
				OperationNotifier: chainmock.OperationNotifier(),
			}

			toSubmit := structs.AttesterSlashingsFromConsensus([]*ethpbv1alpha1.AttesterSlashing{slashing})
			b, err := json.Marshal(toSubmit[0])
			require.NoError(t, err)
			var body bytes.Buffer
			_, err = body.Write(b)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com/beacon/pool/attester_slashings", &body)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttesterSlashings(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			pendingSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, bs, true)
			require.Equal(t, 1, len(pendingSlashings))
			assert.DeepEqual(t, slashing, pendingSlashings[0])
			require.Equal(t, 1, broadcaster.NumMessages())
			assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
			_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.AttesterSlashing)
			assert.Equal(t, true, ok)
		})
		t.Run("invalid-slashing", func(t *testing.T) {
			bs, err := util.NewBeaconState()
			require.NoError(t, err)

			broadcaster := &p2pMock.MockBroadcaster{}
			s := &Server{
				ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
				SlashingsPool:    &slashingsmock.PoolMock{},
				Broadcaster:      broadcaster,
			}

			var body bytes.Buffer
			_, err = body.WriteString(invalidAttesterSlashing)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com/beacon/pool/attester_slashings", &body)
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttesterSlashings(writer, request)
			require.Equal(t, http.StatusBadRequest, writer.Code)
			e := &httputil.DefaultJsonError{}
			require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
			assert.Equal(t, http.StatusBadRequest, e.Code)
			assert.StringContains(t, "Invalid attester slashing", e.Message)
		})
	})
	t.Run("V2", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			attestationData1.Slot = 1
			attestationData2.Slot = 1
			electraSlashing := &ethpbv1alpha1.AttesterSlashingElectra{
				Attestation_1: &ethpbv1alpha1.IndexedAttestationElectra{
					AttestingIndices: []uint64{0},
					Data:             attestationData1,
					Signature:        make([]byte, 96),
				},
				Attestation_2: &ethpbv1alpha1.IndexedAttestationElectra{
					AttestingIndices: []uint64{0},
					Data:             attestationData2,
					Signature:        make([]byte, 96),
				},
			}

			_, keys, err := util.DeterministicDepositsAndKeys(1)
			require.NoError(t, err)
			validator := &ethpbv1alpha1.Validator{
				PublicKey: keys[0].PublicKey().Marshal(),
			}

			ebs, err := util.NewBeaconStateElectra(func(state *ethpbv1alpha1.BeaconStateElectra) error {
				state.Validators = []*ethpbv1alpha1.Validator{validator}
				return nil
			})
			require.NoError(t, err)

			for _, att := range []*ethpbv1alpha1.IndexedAttestationElectra{electraSlashing.Attestation_1, electraSlashing.Attestation_2} {
				sb, err := signing.ComputeDomainAndSign(ebs, att.Data.Target.Epoch, att.Data, params.BeaconConfig().DomainBeaconAttester, keys[0])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				att.Signature = sig.Marshal()
			}

			chainmock := &blockchainmock.ChainService{State: ebs}
			broadcaster := &p2pMock.MockBroadcaster{}
			s := &Server{
				ChainInfoFetcher:  chainmock,
				SlashingsPool:     &slashingsmock.PoolMock{},
				Broadcaster:       broadcaster,
				OperationNotifier: chainmock.OperationNotifier(),
			}

			toSubmit := structs.AttesterSlashingsElectraFromConsensus([]*ethpbv1alpha1.AttesterSlashingElectra{electraSlashing})
			b, err := json.Marshal(toSubmit[0])
			require.NoError(t, err)
			var body bytes.Buffer
			_, err = body.Write(b)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com/beacon/pool/attester_electras", &body)
			request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttesterSlashingsV2(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			pendingSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, ebs, true)
			require.Equal(t, 1, len(pendingSlashings))
			require.Equal(t, 1, broadcaster.NumMessages())
			assert.DeepEqual(t, electraSlashing, pendingSlashings[0])
			assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
			_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.AttesterSlashingElectra)
			assert.Equal(t, true, ok)
		})
		t.Run("accross-fork", func(t *testing.T) {
			attestationData1.Slot = params.BeaconConfig().SlotsPerEpoch
			attestationData2.Slot = params.BeaconConfig().SlotsPerEpoch
			slashing := &ethpbv1alpha1.AttesterSlashingElectra{
				Attestation_1: &ethpbv1alpha1.IndexedAttestationElectra{
					AttestingIndices: []uint64{0},
					Data:             attestationData1,
					Signature:        make([]byte, 96),
				},
				Attestation_2: &ethpbv1alpha1.IndexedAttestationElectra{
					AttestingIndices: []uint64{0},
					Data:             attestationData2,
					Signature:        make([]byte, 96),
				},
			}

			params.SetupTestConfigCleanup(t)
			config := params.BeaconConfig()
			config.AltairForkEpoch = 1
			params.OverrideBeaconConfig(config)

			bs, keys := util.DeterministicGenesisState(t, 1)
			newBs := bs.Copy()
			newBs, err := transition.ProcessSlots(ctx, newBs, params.BeaconConfig().SlotsPerEpoch)
			require.NoError(t, err)

			for _, att := range []*ethpbv1alpha1.IndexedAttestationElectra{slashing.Attestation_1, slashing.Attestation_2} {
				sb, err := signing.ComputeDomainAndSign(newBs, att.Data.Target.Epoch, att.Data, params.BeaconConfig().DomainBeaconAttester, keys[0])
				require.NoError(t, err)
				sig, err := bls.SignatureFromBytes(sb)
				require.NoError(t, err)
				att.Signature = sig.Marshal()
			}

			broadcaster := &p2pMock.MockBroadcaster{}
			chainmock := &blockchainmock.ChainService{State: bs}
			s := &Server{
				ChainInfoFetcher:  chainmock,
				SlashingsPool:     &slashingsmock.PoolMock{},
				Broadcaster:       broadcaster,
				OperationNotifier: chainmock.OperationNotifier(),
			}

			toSubmit := structs.AttesterSlashingsElectraFromConsensus([]*ethpbv1alpha1.AttesterSlashingElectra{slashing})
			b, err := json.Marshal(toSubmit[0])
			require.NoError(t, err)
			var body bytes.Buffer
			_, err = body.Write(b)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "http://example.com/beacon/pool/attester_slashings", &body)
			request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
			writer := httptest.NewRecorder()
			writer.Body = &bytes.Buffer{}

			s.SubmitAttesterSlashingsV2(writer, request)
			require.Equal(t, http.StatusOK, writer.Code)
			pendingSlashings := s.SlashingsPool.PendingAttesterSlashings(ctx, bs, true)
			require.Equal(t, 1, len(pendingSlashings))
			assert.DeepEqual(t, slashing, pendingSlashings[0])
			require.Equal(t, 1, broadcaster.NumMessages())
			assert.Equal(t, true, broadcaster.BroadcastCalled.Load())
			_, ok := broadcaster.BroadcastMessages[0].(*ethpbv1alpha1.AttesterSlashingElectra)
			assert.Equal(t, true, ok)
		})
	})
	t.Run("invalid-slashing", func(t *testing.T) {
		bs, err := util.NewBeaconStateElectra()
		require.NoError(t, err)

		broadcaster := &p2pMock.MockBroadcaster{}
		s := &Server{
			ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
			SlashingsPool:    &slashingsmock.PoolMock{},
			Broadcaster:      broadcaster,
		}

		var body bytes.Buffer
		_, err = body.WriteString(invalidAttesterSlashing)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "http://example.com/beacon/pool/attester_slashings", &body)
		request.Header.Set(api.VersionHeader, version.String(version.Alpaca))
		writer := httptest.NewRecorder()
		writer.Body = &bytes.Buffer{}

		s.SubmitAttesterSlashingsV2(writer, request)
		require.Equal(t, http.StatusBadRequest, writer.Code)
		e := &httputil.DefaultJsonError{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
		assert.Equal(t, http.StatusBadRequest, e.Code)
		assert.StringContains(t, "Invalid attester slashing", e.Message)
	})
}

func TestSubmitProposerSlashing_InvalidSlashing(t *testing.T) {
	bs, err := util.NewBeaconState()
	require.NoError(t, err)

	broadcaster := &p2pMock.MockBroadcaster{}
	s := &Server{
		ChainInfoFetcher: &blockchainmock.ChainService{State: bs},
		SlashingsPool:    &slashingsmock.PoolMock{},
		Broadcaster:      broadcaster,
	}

	var body bytes.Buffer
	_, err = body.WriteString(invalidProposerSlashing)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "http://example.com/beacon/pool/proposer_slashings", &body)
	writer := httptest.NewRecorder()
	writer.Body = &bytes.Buffer{}

	s.SubmitProposerSlashing(writer, request)
	require.Equal(t, http.StatusBadRequest, writer.Code)
	e := &httputil.DefaultJsonError{}
	require.NoError(t, json.Unmarshal(writer.Body.Bytes(), e))
	assert.Equal(t, http.StatusBadRequest, e.Code)
	assert.StringContains(t, "Invalid proposer slashing", e.Message)
}

var (
	singleAtt = `[
  {
    "aggregation_bits": "0x03",
    "signature": "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "target": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      }
    }
  }
]`
	multipleAtts = `[
  {
    "aggregation_bits": "0x03",
    "signature": "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0x736f75726365726f6f7431000000000000000000000000000000000000000000"
      },
      "target": {
        "epoch": "0",
        "root": "0x746172676574726f6f7431000000000000000000000000000000000000000000"
      }
    }
  },
  {
    "aggregation_bits": "0x03",
    "signature": "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0x736f75726365726f6f7431000000000000000000000000000000000000000000"
      },
      "target": {
        "epoch": "0",
        "root": "0x746172676574726f6f7432000000000000000000000000000000000000000000"
      }
    }
  }
]`
	// signature is invalid
	invalidAtt = `[
  {
    "aggregation_bits": "0x03",
    "signature": "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "target": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      }
    }
  }
]`
	singleAttElectra = `[
  {
    "aggregation_bits": "0x03",
	"committee_bits": "0x0100000000000000",
    "signature": "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "target": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      }
    }
  }
]`
	multipleAttsElectra = `[
  {
    "aggregation_bits": "0x03",
	"committee_bits": "0x0100000000000000",
    "signature": "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0x736f75726365726f6f7431000000000000000000000000000000000000000000"
      },
      "target": {
        "epoch": "0",
        "root": "0x746172676574726f6f7431000000000000000000000000000000000000000000"
      }
    }
  },
  {
    "aggregation_bits": "0x03",
	"committee_bits": "0x0100000000000000",
    "signature": "0x8146f4397bfd8fd057ebbcd6a67327bdc7ed5fb650533edcb6377b650dea0b6da64c14ecd60846d5c0a0cd43893d6972092500f82c9d8a955e2b58c5ed3cbe885d84008ace6bd86ba9e23652f58e2ec207cec494c916063257abf285b9b15b15",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0x736f75726365726f6f7431000000000000000000000000000000000000000000"
      },
      "target": {
        "epoch": "0",
        "root": "0x746172676574726f6f7432000000000000000000000000000000000000000000"
      }
    }
  }
]`
	// signature is invalid
	invalidAttElectra = `[
  {
    "aggregation_bits": "0x03",
	"committee_bits": "0x0100000000000000",
	"signature": "0x000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "data": {
      "slot": "0",
      "index": "0",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "target": {
        "epoch": "0",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      }
    }
  }
]`
	exit1 = `{
  "message": {
    "epoch": "0",
    "validator_index": "0"
  },
  "signature": "0xaf20377dabe56887f72273806ea7f3bab3df464fe0178b2ec9bb83d891bf038671c222e2fa7fc0b3e83a0a86ecf235f6104f8130d9e3177cdf5391953fcebb9676f906f4e366b95cb4d734f48f7fc0f116c643519a58a3bb1f7501a1f64b87d2"
}`
	exit2 = fmt.Sprintf(`{
  "message": {
    "epoch": "%d",
    "validator_index": "0"
  },
  "signature": "0x8161b7c8ae6a16c505ccf426de7a920100341f0d968a22c60c475a48dd2a98e88ec42851c7ebda0dcbe5ba751ad31c9e0f4daee9f22373edd4d1439d62f54f803d6d851d2c10f3fb31b4fbe2b4c24e24ff8cb6a79d3f14b0ed5745b5d89abcb3"
}`, params.BeaconConfig().ShardCommitteePeriod+32)
	// epoch is invalid
	invalidExit1 = `{
  "message": {
    "epoch": "foo",
    "validator_index": "0"
  },
  "signature": "0xaf20377dabe56887f72273806ea7f3bab3df464fe0178b2ec9bb83d891bf038671c222e2fa7fc0b3e83a0a86ecf235f6104f8130d9e3177cdf5391953fcebb9676f906f4e366b95cb4d734f48f7fc0f116c643519a58a3bb1f7501a1f64b87d2"
}`
	// signature is wrong
	invalidExit2 = `{
  "message": {
    "epoch": "0",
    "validator_index": "0"
  },
  "signature": "0xa430330829331089c4381427217231c32c26ac551de410961002491257b1ef50c3d49a89fc920ac2f12f0a27a95ab9b811e49f04cb08020ff7dbe03bdb479f85614608c4e5d0108052497f4ae0148c0c2ef79c05adeaf74e6c003455f2cc5716"
}`
	// non-existing validator index
	invalidExit3 = `{
  "message": {
    "epoch": "0",
    "validator_index": "99"
  },
  "signature": "0xa430330829331089c4381427217231c32c26ac551de410961002491257b1ef50c3d49a89fc920ac2f12f0a27a95ab9b811e49f04cb08020ff7dbe03bdb479f85614608c4e5d0108052497f4ae0148c0c2ef79c05adeaf74e6c003455f2cc5716"
}`
	// signatures are invalid
	invalidAttesterSlashing = `{
  "attestation_1": {
    "attesting_indices": [
      "1"
    ],
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
    "data": {
      "slot": "1",
      "index": "1",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "1",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "target": {
        "epoch": "1",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      }
    }
  },
  "attestation_2": {
    "attesting_indices": [
      "1"
    ],
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505",
    "data": {
      "slot": "1",
      "index": "1",
      "beacon_block_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "source": {
        "epoch": "1",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      },
      "target": {
        "epoch": "1",
        "root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
      }
    }
  }
}`
	// signatures are invalid
	invalidProposerSlashing = `{
  "signed_header_1": {
    "message": {
      "slot": "1",
      "proposer_index": "1",
      "parent_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "state_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "body_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  },
  "signed_header_2": {
    "message": {
      "slot": "1",
      "proposer_index": "1",
      "parent_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "state_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
      "body_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
    },
    "signature": "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
  }
}`
)

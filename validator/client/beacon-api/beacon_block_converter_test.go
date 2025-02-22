package beacon_api

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	testhelpers "github.com/prysmaticlabs/prysm/v5/validator/client/beacon-api/test-helpers"
)

func TestGetBeaconBlockConverter_Phase0Valid(t *testing.T) {
	expectedBeaconBlock := testhelpers.GenerateProtoPhase0BeaconBlock()
	beaconBlockConverter := &beaconApiBeaconBlockConverter{}
	beaconBlock, err := beaconBlockConverter.ConvertRESTPhase0BlockToProto(testhelpers.GenerateJsonPhase0BeaconBlock())
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlockConverter_Phase0Error(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *structs.BeaconBlock
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "nil eth1 data",
			expectedErrorMessage: "eth1 data is nil",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad slot",
			expectedErrorMessage: "failed to parse slot `foo`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Slot = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad proposer index",
			expectedErrorMessage: "failed to parse proposer index `bar`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.ProposerIndex = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad parent root",
			expectedErrorMessage: "failed to decode parent root `foo`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.ParentRoot = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad state root",
			expectedErrorMessage: "failed to decode state root `bar`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.StateRoot = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad randao reveal",
			expectedErrorMessage: "failed to decode randao reveal `foo`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.RandaoReveal = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad deposit root",
			expectedErrorMessage: "failed to decode deposit root `bar`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data.DepositRoot = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad deposit count",
			expectedErrorMessage: "failed to parse deposit count `foo`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data.DepositCount = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad block hash",
			expectedErrorMessage: "failed to decode block hash `bar`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Eth1Data.BlockHash = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad graffiti",
			expectedErrorMessage: "failed to decode graffiti `foo`",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Graffiti = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad proposer slashings",
			expectedErrorMessage: "failed to get proposer slashings",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.ProposerSlashings[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad attester slashings",
			expectedErrorMessage: "failed to get attester slashings",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.AttesterSlashings[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad attestations",
			expectedErrorMessage: "failed to get attestations",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Attestations[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad deposits",
			expectedErrorMessage: "failed to get deposits",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.Deposits[0] = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad voluntary exits",
			expectedErrorMessage: "failed to get voluntary exits",
			generateData: func() *structs.BeaconBlock {
				beaconBlock := testhelpers.GenerateJsonPhase0BeaconBlock()
				beaconBlock.Body.VoluntaryExits[0] = nil
				return beaconBlock
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			beaconBlockJson := testCase.generateData()

			beaconBlockConverter := &beaconApiBeaconBlockConverter{}
			_, err := beaconBlockConverter.ConvertRESTPhase0BlockToProto(beaconBlockJson)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func TestGetBeaconBlockConverter_AltairValid(t *testing.T) {
	expectedBeaconBlock := testhelpers.GenerateProtoAltairBeaconBlock()
	beaconBlockConverter := &beaconApiBeaconBlockConverter{}
	beaconBlock, err := beaconBlockConverter.ConvertRESTAltairBlockToProto(testhelpers.GenerateJsonAltairBeaconBlock())
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlockConverter_AltairError(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *structs.BeaconBlockAltair
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *structs.BeaconBlockAltair {
				beaconBlock := testhelpers.GenerateJsonAltairBeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad phase0 fields",
			expectedErrorMessage: "failed to get the phase0 fields of the altair block",
			generateData: func() *structs.BeaconBlockAltair {
				beaconBlock := testhelpers.GenerateJsonAltairBeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			beaconBlockJson := testCase.generateData()

			beaconBlockConverter := &beaconApiBeaconBlockConverter{}
			_, err := beaconBlockConverter.ConvertRESTAltairBlockToProto(beaconBlockJson)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func TestGetBeaconBlockConverter_BellatrixValid(t *testing.T) {
	expectedBeaconBlock := testhelpers.GenerateProtoBellatrixBeaconBlock()
	beaconBlockConverter := &beaconApiBeaconBlockConverter{}
	beaconBlock, err := beaconBlockConverter.ConvertRESTBellatrixBlockToProto(testhelpers.GenerateJsonBellatrixBeaconBlock())
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlockConverter_BellatrixError(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *structs.BeaconBlockBellatrix
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "nil execution payload",
			expectedErrorMessage: "execution payload is nil",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad altair fields",
			expectedErrorMessage: "failed to get the altair fields of the bellatrix block",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad parent hash",
			expectedErrorMessage: "failed to decode execution payload parent hash `foo`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.ParentHash = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad fee recipient",
			expectedErrorMessage: "failed to decode execution payload fee recipient `bar`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.FeeRecipient = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad state root",
			expectedErrorMessage: "failed to decode execution payload state root `foo`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.StateRoot = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad receipts root",
			expectedErrorMessage: "failed to decode execution payload receipts root `bar`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.ReceiptsRoot = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad logs bloom",
			expectedErrorMessage: "failed to decode execution payload logs bloom `foo`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.LogsBloom = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad prev randao",
			expectedErrorMessage: "failed to decode execution payload prev randao `bar`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.PrevRandao = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad block number",
			expectedErrorMessage: "failed to parse execution payload block number `foo`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.BlockNumber = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad gas limit",
			expectedErrorMessage: "failed to parse execution payload gas limit `bar`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.GasLimit = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad gas used",
			expectedErrorMessage: "failed to parse execution payload gas used `foo`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.GasUsed = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad timestamp",
			expectedErrorMessage: "failed to parse execution payload timestamp `bar`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.Timestamp = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad extra data",
			expectedErrorMessage: "failed to decode execution payload extra data `foo`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.ExtraData = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad base fee per gas",
			expectedErrorMessage: "failed to parse execution payload base fee per gas `bar`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.BaseFeePerGas = "bar"
				return beaconBlock
			},
		},
		{
			name:                 "bad block hash",
			expectedErrorMessage: "failed to decode execution payload block hash `foo`",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.BlockHash = "foo"
				return beaconBlock
			},
		},
		{
			name:                 "bad transactions",
			expectedErrorMessage: "failed to get execution payload transactions",
			generateData: func() *structs.BeaconBlockBellatrix {
				beaconBlock := testhelpers.GenerateJsonBellatrixBeaconBlock()
				beaconBlock.Body.ExecutionPayload.Transactions[0] = "bar"
				return beaconBlock
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			beaconBlockJson := testCase.generateData()

			beaconBlockConverter := &beaconApiBeaconBlockConverter{}
			_, err := beaconBlockConverter.ConvertRESTBellatrixBlockToProto(beaconBlockJson)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

func TestGetBeaconBlockConverter_CapellaValid(t *testing.T) {
	expectedBeaconBlock := testhelpers.GenerateProtoCapellaBeaconBlock()
	beaconBlockConverter := &beaconApiBeaconBlockConverter{}
	beaconBlock, err := beaconBlockConverter.ConvertRESTCapellaBlockToProto(testhelpers.GenerateJsonCapellaBeaconBlock())
	require.NoError(t, err)
	assert.DeepEqual(t, expectedBeaconBlock, beaconBlock)
}

func TestGetBeaconBlockConverter_CapellaError(t *testing.T) {
	testCases := []struct {
		name                 string
		expectedErrorMessage string
		generateData         func() *structs.BeaconBlockCapella
	}{
		{
			name:                 "nil body",
			expectedErrorMessage: "block body is nil",
			generateData: func() *structs.BeaconBlockCapella {
				beaconBlock := testhelpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body = nil
				return beaconBlock
			},
		},
		{
			name:                 "nil execution payload",
			expectedErrorMessage: "execution payload is nil",
			generateData: func() *structs.BeaconBlockCapella {
				beaconBlock := testhelpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body.ExecutionPayload = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad bellatrix fields",
			expectedErrorMessage: "failed to get the bellatrix fields of the capella block",
			generateData: func() *structs.BeaconBlockCapella {
				beaconBlock := testhelpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body.Eth1Data = nil
				return beaconBlock
			},
		},
		{
			name:                 "bad withdrawals",
			expectedErrorMessage: "failed to get withdrawals",
			generateData: func() *structs.BeaconBlockCapella {
				beaconBlock := testhelpers.GenerateJsonCapellaBeaconBlock()
				beaconBlock.Body.ExecutionPayload.Withdrawals[0] = nil
				return beaconBlock
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			beaconBlockJson := testCase.generateData()

			beaconBlockConverter := &beaconApiBeaconBlockConverter{}
			_, err := beaconBlockConverter.ConvertRESTCapellaBlockToProto(beaconBlockJson)
			assert.ErrorContains(t, testCase.expectedErrorMessage, err)
		})
	}
}

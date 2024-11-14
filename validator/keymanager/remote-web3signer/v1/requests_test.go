package v1_test

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	validatorpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	v1 "github.com/prysmaticlabs/prysm/v5/validator/keymanager/remote-web3signer/v1"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager/remote-web3signer/v1/mock"
)

func TestGetAggregateAndProofSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.AggregateAndProofSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request:               mock.GetMockSignRequest("AGGREGATE_AND_PROOF"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    mock.AggregateAndProofSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetAggregateAndProofSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAggregateAndProofSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAggregateAndProofSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAggregationSlotSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.AggregationSlotSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request:               mock.GetMockSignRequest("AGGREGATION_SLOT"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    mock.AggregationSlotSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetAggregationSlotSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAggregationSlotSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAggregationSlotSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAttestationSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.AttestationSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request:               mock.GetMockSignRequest("ATTESTATION"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: mock.AttestationSignRequest(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetAttestationSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAttestationSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAttestationSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBlockSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.BlockSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    mock.BlockSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetBlockSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBlockV2AltairSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.BlockAltairSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK_V2"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    mock.BlockV2AltairSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetBlockAltairSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockAltairSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockAltairSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRandaoRevealSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.RandaoRevealSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request:               mock.GetMockSignRequest("RANDAO_REVEAL"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    mock.RandaoRevealSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetRandaoRevealSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRandaoRevealSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRandaoRevealSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVoluntaryExitSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.VoluntaryExitSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request:               mock.GetMockSignRequest("VOLUNTARY_EXIT"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want:    mock.VoluntaryExitSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetVoluntaryExitSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVoluntaryExitSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVoluntaryExitSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBlockV2BlindedSignRequest(t *testing.T) {
	type args struct {
		request               *validatorpb.SignRequest
		genesisValidatorsRoot []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.BlockV2BlindedSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test non blinded Bellatrix",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK_V2_BELLATRIX"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: mock.BlockV2BlindedSignRequest(func(t *testing.T) []byte {
				bytevalue, err := hexutil.Decode("0xd0a47fef2c05f04dca00e36f0c74c9b5910a30dd4442b6c21ec2befa840fd773")
				require.NoError(t, err)
				return bytevalue
			}(t), "BELLATRIX"),
			wantErr: false,
		},
		{
			name: "Happy Path Test blinded Bellatrix",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK_V2_BLINDED_BELLATRIX"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: mock.BlockV2BlindedSignRequest(func(t *testing.T) []byte {
				bytevalue, err := hexutil.Decode("0x38655b6cae88cbc2d8978287a0f316b19a6b3c2969cfb87ddc25d76955e12b86")
				require.NoError(t, err)
				return bytevalue
			}(t), "BELLATRIX"),
			wantErr: false,
		},
		{
			name: "Happy Path Test non blinded Capella",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK_V2_CAPELLA"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: mock.BlockV2BlindedSignRequest(func(t *testing.T) []byte {
				bytevalue, err := hexutil.Decode("0x7af02a5d0e7ec0aca15cc58fb4132f44806da1b429b8ce4092e05dced9802400")
				require.NoError(t, err)
				return bytevalue
			}(t), "CAPELLA"),
			wantErr: false,
		},
		{
			name: "Happy Path Test blinded Capella",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK_V2_BLINDED_CAPELLA"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: mock.BlockV2BlindedSignRequest(func(t *testing.T) []byte {
				bytevalue, err := hexutil.Decode("0x38655b6cae88cbc2d8978287a0f316b19a6b3c2969cfb87ddc25d76955e12b86")
				require.NoError(t, err)
				return bytevalue
			}(t), "CAPELLA"),
			wantErr: false,
		},
		{
			name: "Happy Path Test non blinded Deneb",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK_V2_DENEB"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: mock.BlockV2BlindedSignRequest(func(t *testing.T) []byte {
				bytevalue, err := hexutil.Decode("0xdde84ef72700c644c8a16c7f65ebb4d88c1ec436368683a7e692166bc1c7a10f")
				require.NoError(t, err)
				return bytevalue
			}(t), "DENEB"),
			wantErr: false,
		},
		{
			name: "Happy Path Test blinded Deneb",
			args: args{
				request:               mock.GetMockSignRequest("BLOCK_V2_BLINDED_DENEB"),
				genesisValidatorsRoot: make([]byte, fieldparams.RootLength),
			},
			want: mock.BlockV2BlindedSignRequest(func(t *testing.T) []byte {
				bytevalue, err := hexutil.Decode("0xd047cdc0090caaad022de8244779e0e81fc52638982834fe6cdfacc692ad6cff")
				require.NoError(t, err)
				return bytevalue
			}(t), "DENEB"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetBlockV2BlindedSignRequest(tt.args.request, tt.args.genesisValidatorsRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlockV2BlindedSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBlockV2BlindedSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValidatorRegistrationSignRequest(t *testing.T) {
	type args struct {
		request *validatorpb.SignRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *v1.ValidatorRegistrationSignRequest
		wantErr bool
	}{
		{
			name: "Happy Path Test",
			args: args{
				request: mock.GetMockSignRequest("VALIDATOR_REGISTRATION"),
			},
			want:    mock.ValidatorRegistrationSignRequest(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v1.GetValidatorRegistrationSignRequest(tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValidatorRegistrationSignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValidatorRegistrationSignRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

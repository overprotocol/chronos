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
				bytevalue, err := hexutil.Decode("0x243456956e64f28981e8db4a7295983457483b2e32468ce25c250436c4564778")
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
				bytevalue, err := hexutil.Decode("0xc6c3aad34578ee1c15cd80a5c19282aae6ba257bffbc3679906e3982c6435d4b")
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
				bytevalue, err := hexutil.Decode("0x0821c10dde580c3ca6f63384f9aea4936fc8deb901e0a2837cb0a0f1737fd51a")
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
				bytevalue, err := hexutil.Decode("0xc6c3aad34578ee1c15cd80a5c19282aae6ba257bffbc3679906e3982c6435d4b")
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
				bytevalue, err := hexutil.Decode("0x6e19ee9f6655ad31a589579fbd515e7c62e991c9d7fb31d2544e33f08216a3db")
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
				bytevalue, err := hexutil.Decode("0xe645d23e36fc0d0d71a90299884013668249c4208660a3b1f1db8ccd3c2a7b9e")
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

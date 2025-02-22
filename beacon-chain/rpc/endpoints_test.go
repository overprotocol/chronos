package rpc

import (
	"net/http"
	"path/filepath"
	"slices"
	"testing"

	"github.com/prysmaticlabs/prysm/v5/api"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"golang.org/x/exp/maps"
)

func Test_endpoints(t *testing.T) {
	rewardsRoutes := map[string][]string{
		"/eth/v1/beacon/rewards/blocks/{block_id}":    {http.MethodGet},
		"/eth/v1/beacon/rewards/attestations/{epoch}": {http.MethodPost},
	}

	beaconRoutes := map[string][]string{
		"/eth/v1/beacon/genesis":                                     {http.MethodGet},
		"/eth/v1/beacon/states/{state_id}/root":                      {http.MethodGet},
		"/eth/v1/beacon/states/{state_id}/fork":                      {http.MethodGet},
		"/eth/v1/beacon/states/{state_id}/finality_checkpoints":      {http.MethodGet},
		"/eth/v1/beacon/states/{state_id}/validators":                {http.MethodGet, http.MethodPost},
		"/eth/v1/beacon/states/{state_id}/validators/{validator_id}": {http.MethodGet},
		"/eth/v1/beacon/states/{state_id}/validator_balances":        {http.MethodGet, http.MethodPost},
		"/eth/v1/beacon/states/{state_id}/committees":                {http.MethodGet},
		"/eth/v1/beacon/states/{state_id}/randao":                    {http.MethodGet},
		"/eth/v1/beacon/headers":                                     {http.MethodGet},
		"/eth/v1/beacon/headers/{block_id}":                          {http.MethodGet},
		"/eth/v1/beacon/blinded_blocks":                              {http.MethodPost},
		"/eth/v2/beacon/blinded_blocks":                              {http.MethodPost},
		"/eth/v1/beacon/blocks":                                      {http.MethodPost},
		"/eth/v2/beacon/blocks":                                      {http.MethodPost},
		"/eth/v2/beacon/blocks/{block_id}":                           {http.MethodGet},
		"/eth/v1/beacon/blocks/{block_id}/root":                      {http.MethodGet},
		"/eth/v1/beacon/blocks/{block_id}/attestations":              {http.MethodGet},
		"/eth/v2/beacon/blocks/{block_id}/attestations":              {http.MethodGet},
		"/eth/v1/beacon/blob_sidecars/{block_id}":                    {http.MethodGet},
		"/eth/v1/beacon/deposit_snapshot":                            {http.MethodGet},
		"/eth/v1/beacon/blinded_blocks/{block_id}":                   {http.MethodGet},
		"/eth/v1/beacon/pool/attestations":                           {http.MethodGet, http.MethodPost},
		"/eth/v2/beacon/pool/attestations":                           {http.MethodGet, http.MethodPost},
		"/eth/v1/beacon/pool/attester_slashings":                     {http.MethodGet, http.MethodPost},
		"/eth/v2/beacon/pool/attester_slashings":                     {http.MethodGet, http.MethodPost},
		"/eth/v1/beacon/pool/proposer_slashings":                     {http.MethodGet, http.MethodPost},
		"/eth/v1/beacon/pool/voluntary_exits":                        {http.MethodGet, http.MethodPost},
		"/prysm/v1/beacon/individual_votes":                          {http.MethodPost},
	}

	builderRoutes := map[string][]string{
		"/eth/v1/builder/states/{state_id}/expected_withdrawals": {http.MethodGet},
	}

	blobRoutes := map[string][]string{
		"/eth/v1/beacon/blob_sidecars/{block_id}": {http.MethodGet},
	}

	configRoutes := map[string][]string{
		"/eth/v1/config/fork_schedule":    {http.MethodGet},
		"/eth/v1/config/spec":             {http.MethodGet},
		"/eth/v1/config/deposit_contract": {http.MethodGet},
	}

	debugRoutes := map[string][]string{
		"/eth/v2/debug/beacon/states/{state_id}": {http.MethodGet},
		"/eth/v2/debug/beacon/heads":             {http.MethodGet},
		"/eth/v1/debug/fork_choice":              {http.MethodGet},
	}

	eventsRoutes := map[string][]string{
		"/eth/v1/events": {http.MethodGet},
	}

	nodeRoutes := map[string][]string{
		"/eth/v1/node/identity":        {http.MethodGet},
		"/eth/v1/node/peers":           {http.MethodGet},
		"/eth/v1/node/peers/{peer_id}": {http.MethodGet},
		"/eth/v1/node/peer_count":      {http.MethodGet},
		"/eth/v1/node/version":         {http.MethodGet},
		"/eth/v1/node/syncing":         {http.MethodGet},
		"/eth/v1/node/health":          {http.MethodGet},
	}

	validatorRoutes := map[string][]string{
		"/eth/v1/validator/duties/attester/{epoch}":        {http.MethodPost},
		"/eth/v1/validator/duties/proposer/{epoch}":        {http.MethodGet},
		"/eth/v2/validator/blocks/{slot}":                  {http.MethodGet},
		"/eth/v3/validator/blocks/{slot}":                  {http.MethodGet},
		"/eth/v1/validator/blinded_blocks/{slot}":          {http.MethodGet},
		"/eth/v1/validator/attestation_data":               {http.MethodGet},
		"/eth/v1/validator/aggregate_attestation":          {http.MethodGet},
		"/eth/v2/validator/aggregate_attestation":          {http.MethodGet},
		"/eth/v1/validator/aggregate_and_proofs":           {http.MethodPost},
		"/eth/v2/validator/aggregate_and_proofs":           {http.MethodPost},
		"/eth/v1/validator/beacon_committee_subscriptions": {http.MethodPost},
		"/eth/v1/validator/beacon_committee_selections":    {http.MethodPost},
		"/eth/v1/validator/prepare_beacon_proposer":        {http.MethodPost},
		"/eth/v1/validator/register_validator":             {http.MethodPost},
		"/eth/v1/validator/liveness/{epoch}":               {http.MethodPost},
	}

	prysmBeaconRoutes := map[string][]string{
		"/prysm/v1/beacon/weak_subjectivity":                 {http.MethodGet},
		"/eth/v1/beacon/states/{state_id}/validator_count":   {http.MethodGet},
		"/prysm/v1/beacon/states/{state_id}/validator_count": {http.MethodGet},
		"/prysm/v1/beacon/chain_head":                        {http.MethodGet},
		"/prysm/v1/beacon/blobs":                             {http.MethodPost},
	}

	prysmNodeRoutes := map[string][]string{
		"/prysm/node/trusted_peers":              {http.MethodGet, http.MethodPost},
		"/prysm/v1/node/trusted_peers":           {http.MethodGet, http.MethodPost},
		"/prysm/node/trusted_peers/{peer_id}":    {http.MethodDelete},
		"/prysm/v1/node/trusted_peers/{peer_id}": {http.MethodDelete},
	}

	prysmValidatorRoutes := map[string][]string{
		"/prysm/validators/performance":           {http.MethodPost},
		"/prysm/v1/validators/performance":        {http.MethodPost},
		"/prysm/v1/validators/participation":      {http.MethodGet},
		"/prysm/v1/validators/active_set_changes": {http.MethodGet},
	}

	overRoutes := map[string][]string{
		"/chronos/states/epoch_reward/{epoch}":                                   {http.MethodGet},
		"/over/v1/beacon/states/{state_id}/reserves":                             {http.MethodGet},
		"/over/v1/beacon/states/{state_id}/deposit_estimation":                   {http.MethodGet},
		"/over/v1/beacon/states/{state_id}/deposit_estimation/{pubkey}":          {http.MethodGet},
		"/over/v1/beacon/states/{state_id}/withdrawal_estimation/{validator_id}": {http.MethodGet},
		"/over/v1/beacon/states/{state_id}/exit/queue_epoch":                     {http.MethodGet},
	}

	overNodeRoutes := map[string][]string{
		"/over-node/close": {http.MethodPost},
	}

	s := &Service{cfg: &Config{
		AuthTokenPath: filepath.Join(t.TempDir(), "auth_token"),
	}}

	endpoints := s.endpoints(true, true, nil, nil, nil, nil, nil, nil, nil)
	actualRoutes := make(map[string][]string, len(endpoints))
	for _, e := range endpoints {
		if _, ok := actualRoutes[e.template]; ok {
			actualRoutes[e.template] = append(actualRoutes[e.template], e.methods...)
		} else {
			actualRoutes[e.template] = e.methods
		}
	}
	expectedRoutes := combineMaps(beaconRoutes, builderRoutes, configRoutes, debugRoutes, eventsRoutes, nodeRoutes, validatorRoutes, rewardsRoutes, blobRoutes, prysmValidatorRoutes, prysmNodeRoutes, prysmBeaconRoutes, overRoutes, overNodeRoutes)

	assert.Equal(t, true, maps.EqualFunc(expectedRoutes, actualRoutes, func(actualMethods []string, expectedMethods []string) bool {
		return slices.Equal(expectedMethods, actualMethods)
	}))
}

func Test_overNodeRoutes(t *testing.T) {
	tests := []struct {
		name          string
		authTokenPath string
		prepare       func(authTokenPath string) error
		expectedCount int
	}{
		{
			name:          "No auth token path is provided",
			authTokenPath: "",
			prepare:       nil,
			expectedCount: 0,
		},
		{
			name:          "Auth token path is provided (empty)",
			authTokenPath: filepath.Join(t.TempDir(), "auth_token"),
			prepare:       nil, // No preparation needed for an empty file
			expectedCount: 1,
		},
		{
			name:          "Auth token path is provided (non-empty)",
			authTokenPath: filepath.Join(t.TempDir(), "auth_token"),
			prepare: func(authTokenPath string) error {
				token, err := api.GenerateRandomHexString()
				if err != nil {
					return err
				}
				return saveAuthToken(authTokenPath, token)
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepare != nil {
				require.NoError(t, tt.prepare(tt.authTokenPath))
			}

			s := &Service{cfg: &Config{
				AuthTokenPath: tt.authTokenPath,
			}}
			endpoints := s.overNodeEndpoints(nil)
			require.Equal(t, tt.expectedCount, len(endpoints))
		})
	}
}

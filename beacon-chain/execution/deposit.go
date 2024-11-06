package execution

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/params"
)

// DepositContractAddress returns the deposit contract address for the given chain.
func DepositContractAddress() (string, error) {
	address := params.BeaconConfig().DepositContractAddress
	if address == "" {
		return "", errors.New("valid deposit contract is required")
	}

	if !common.IsHexAddress(address) {
		return "", errors.New("invalid deposit contract address given: " + address)
	}
	return address, nil
}

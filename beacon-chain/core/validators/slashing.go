package validators

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

// SlashingParamsPerVersion returns the slashing parameters for the given state version.
func SlashingParamsPerVersion(v int) (slashingQuotient, proposerRewardQuotient, whistleblowerRewardQuotient uint64, err error) {
	cfg := params.BeaconConfig()

	if v >= version.Alpaca {
		slashingQuotient = cfg.MinSlashingPenaltyQuotientAlpaca
		proposerRewardQuotient = cfg.ProposerRewardQuotient
		whistleblowerRewardQuotient = cfg.WhistleBlowerRewardQuotientAlpaca
		return
	}

	if v >= version.Bellatrix {
		slashingQuotient = cfg.MinSlashingPenaltyQuotientBellatrix
		proposerRewardQuotient = cfg.ProposerRewardQuotient
		whistleblowerRewardQuotient = cfg.WhistleBlowerRewardQuotient
		return
	}

	if v >= version.Altair {
		slashingQuotient = cfg.MinSlashingPenaltyQuotientAltair
		proposerRewardQuotient = cfg.ProposerRewardQuotient
		whistleblowerRewardQuotient = cfg.WhistleBlowerRewardQuotient
		return
	}

	if v >= version.Phase0 {
		slashingQuotient = cfg.MinSlashingPenaltyQuotient
		proposerRewardQuotient = cfg.ProposerRewardQuotient
		whistleblowerRewardQuotient = cfg.WhistleBlowerRewardQuotient
		return
	}

	err = errors.Errorf("unknown state version %s", version.String(v))
	return
}

package eth

type copier[T any] interface {
	Copy() T
}

func CopySlice[T any, C copier[T]](original []C) []T {
	// Create a new slice with the same length as the original
	newSlice := make([]T, len(original))
	for i := 0; i < len(newSlice); i++ {
		newSlice[i] = original[i].Copy()
	}
	return newSlice
}

// CopyValidator copies the provided validator.
func CopyValidator(val *Validator) *Validator {
	pubKey := make([]byte, len(val.PublicKey))
	copy(pubKey, val.PublicKey)
	withdrawalCreds := make([]byte, len(val.WithdrawalCredentials))
	copy(withdrawalCreds, val.WithdrawalCredentials)
	return &Validator{
		PublicKey:                  pubKey,
		WithdrawalCredentials:      withdrawalCreds,
		EffectiveBalance:           val.EffectiveBalance,
		Slashed:                    val.Slashed,
		ActivationEligibilityEpoch: val.ActivationEligibilityEpoch,
		ActivationEpoch:            val.ActivationEpoch,
		ExitEpoch:                  val.ExitEpoch,
		WithdrawableEpoch:          val.WithdrawableEpoch,
		PrincipalBalance:           val.PrincipalBalance,
	}
}

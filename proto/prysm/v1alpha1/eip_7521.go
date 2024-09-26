package eth

// Copy --
func (pw *PendingPartialWithdrawal) Copy() *PendingPartialWithdrawal {
	if pw == nil {
		return nil
	}
	return &PendingPartialWithdrawal{
		Index:             pw.Index,
		Amount:            pw.Amount,
		WithdrawableEpoch: pw.WithdrawableEpoch,
	}
}

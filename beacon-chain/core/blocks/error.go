package blocks

import "github.com/pkg/errors"

var errInvalidBLSPrefix = errors.New("withdrawal credential prefix is not a BLS prefix")
var errInvalidWithdrawalCredentials = errors.New("withdrawal credentials do not match")

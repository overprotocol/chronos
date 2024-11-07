package execution

import (
	"strings"

	"github.com/pkg/errors"
)

const defaultEth1HeaderReqLimit = uint64(1000)

var errTimedOut = errors.New("net/http: request canceled")

func clientTimedOutError(err error) bool {
	return strings.Contains(err.Error(), errTimedOut.Error())
}

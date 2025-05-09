package utils

import (
	"fmt"
	"sync/atomic"
)

var sessionCounter uint32 = 1000

func GenerateSessionID() string {
	return fmt.Sprintf("%d", atomic.AddUint32(&sessionCounter, 1))
}

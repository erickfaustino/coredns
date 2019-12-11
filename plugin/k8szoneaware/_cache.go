package k8szoneaware

import (
	"sync"
)

type ip map[string]string

var cache = make(map[string]ip)

var mutex = &sync.Mutex{}

func Set(pod, region, ip string) string {
	mutex.Lock()
	ippod := map[string]string{region: ip}
	cache[pod] = ippod
	mutex.Unlock()
}

package extfmt

import (
	"fmt"
	"sync"
)

var reg *Registry

func init() {
	reg = &Registry{
		implementations:      map[string]ExtFmt{},
		implementationsMutex: &sync.Mutex{},
	}
}

type Registry struct {
	implementations      map[string]ExtFmt
	implementationsMutex *sync.Mutex
}

func RegisterExtFmt(key string, impl ExtFmt) {
	reg.implementationsMutex.Lock()
	defer reg.implementationsMutex.Unlock()
	if key == "" {
		panic("invalid extfmt implementation key")
	}
	if _, exists := reg.implementations[key]; exists {
		panic(fmt.Sprintf("cannot register duplicate extfmt implementation with key: %s", key))
	}
	reg.implementations[key] = impl
}

func GetImplementation(key string) ExtFmt {
	reg.implementationsMutex.Lock()
	defer reg.implementationsMutex.Unlock()
	return reg.implementations[key]
}

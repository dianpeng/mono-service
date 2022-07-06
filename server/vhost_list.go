package server

import (
	"fmt"
	"github.com/dianpeng/mono-service/vhost"
	"sync"
)

// This is the naive implementation of a thread safe vhost list. The vhostlist
// owned by each listener is of critical importance to be used for configuration
// update, add, remove etc ...
type vhostlist struct {
	// Notes, when an index is pointed to something, then the vhost must be existed
	index map[string]*vhost.VHost
	name  map[string]*vhost.VHost
	lock  sync.RWMutex
}

func newvhostlist() vhostlist {
	return vhostlist{
		index: make(map[string]*vhost.VHost),
		name:  make(map[string]*vhost.VHost),
	}
}

func (v *vhostlist) add(
	vhost *vhost.VHost,
) error {
	serverName := vhost.Config.ServerName
	vhostName := vhost.Config.Name

	// (0) try to get it from the serverNameIndex if needed.
	v.lock.Lock()
	defer v.lock.Unlock()

	{
		_, ok := v.index[serverName]
		if ok {
			return fmt.Errorf("server name %s already existed", serverName)
		}
	}
	{
		_, ok := v.name[vhostName]
		if ok {
			return fmt.Errorf("vhost name %s already existed", vhostName)
		}
	}

	v.index[serverName] = vhost
	v.name[vhostName] = vhost
	return nil
}

func (v *vhostlist) remove(
	vhostName string,
) bool {
	v.lock.Lock()
	defer v.lock.Unlock()
	val, ok := v.name[vhostName]
	if !ok {
		return false
	}
	delete(v.index, val.Config.ServerName)
	delete(v.name, vhostName)
	return true
}

func (v *vhostlist) update(
	vhost *vhost.VHost,
) {
	serverName := vhost.Config.ServerName
	vhostName := vhost.Config.Name

	// make the name and index table consistent
	v.remove(vhostName)

	v.lock.Lock()
	defer v.lock.Unlock()

	v.index[serverName] = vhost
	v.name[vhostName] = vhost
}

func (v *vhostlist) get(
	vhostName string,
) *vhost.VHost {
	v.lock.Lock()
	defer v.lock.Unlock()
	val, ok := v.name[vhostName]
	if ok {
		return val
	} else {
		return nil
	}
}

func (v *vhostlist) resolve(
	host string,
) *vhost.VHost {
	v.lock.Lock()
	defer v.lock.Unlock()
	val, ok := v.index[host]
	if ok {
		return val
	} else {
		return nil
	}
}

package routing

import "sync"

type PrefixTable struct {
	V map[string]*TableEntry
	Mux sync.RWMutex
}

type TableEntry struct {
	SequenceNumber uint32
	NextHop string
}

// returns the next hop for the given destination name
// if the given destination name is does not exists, empty string is returned
func (table *PrefixTable) GetPrefix(destinationName string) string {
	table.Mux.RLock()
	defer table.Mux.RUnlock()
	return table.V[destinationName].NextHop
}

// updates the prefix table of the given destination name
//
func (table *PrefixTable) UpdatePrefix(destinationName, newAddress string) {

}
package bgptools

import "net/netip"

type trieNode struct {
	zero *trieNode
	one  *trieNode
	data *prefixRecord
}

func newTrie() *trieNode {
	return &trieNode{}
}

func insertPrefix(root *trieNode, prefix netip.Prefix, data prefixRecord) {
	if root == nil {
		return
	}

	prefix = prefix.Masked()
	addr := prefix.Addr().Unmap()
	bits := prefix.Bits()

	node := root

	if addr.Is4() {
		bytes := addr.As4()
		for idx := 0; idx < bits && idx < 32; idx++ {
			bit := bitAt(bytes[:], idx)
			if bit == 0 {
				if node.zero == nil {
					node.zero = &trieNode{}
				}
				node = node.zero
				continue
			}

			if node.one == nil {
				node.one = &trieNode{}
			}
			node = node.one
		}
		node.data = &data
		return
	}

	bytes := addr.As16()
	for idx := 0; idx < bits && idx < 128; idx++ {
		bit := bitAt(bytes[:], idx)
		if bit == 0 {
			if node.zero == nil {
				node.zero = &trieNode{}
			}
			node = node.zero
			continue
		}

		if node.one == nil {
			node.one = &trieNode{}
		}
		node = node.one
	}
	node.data = &data
}

func longestMatch(root *trieNode, addr netip.Addr) *prefixRecord {
	if root == nil {
		return nil
	}

	addr = addr.Unmap()

	node := root
	var best *prefixRecord

	if addr.Is4() {
		bytes := addr.As4()
		for idx := 0; idx < 32; idx++ {
			if node == nil {
				break
			}
			if node.data != nil {
				best = node.data
			}

			if bitAt(bytes[:], idx) == 0 {
				node = node.zero
			} else {
				node = node.one
			}
		}
		if node != nil && node.data != nil {
			best = node.data
		}
		return best
	}

	bytes := addr.As16()
	for idx := 0; idx < 128; idx++ {
		if node == nil {
			break
		}
		if node.data != nil {
			best = node.data
		}

		if bitAt(bytes[:], idx) == 0 {
			node = node.zero
		} else {
			node = node.one
		}
	}
	if node != nil && node.data != nil {
		best = node.data
	}

	return best
}

func bitAt(raw []byte, index int) int {
	if index < 0 {
		return 0
	}
	byteIndex := index / 8
	if byteIndex >= len(raw) {
		return 0
	}
	bitOffset := uint(7 - (index % 8))
	if ((raw[byteIndex] >> bitOffset) & 1) == 1 {
		return 1
	}
	return 0
}

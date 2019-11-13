package syncdatasources

import "hash/adler32"

// Hash for given string 'str' calculate hash value and then transform it into [0, nodeNum) number
// If nodeNum matches nodeIdx then hash is correct for this node, otherwise it isn't
func Hash(str string, nodeIdx, nodeNum int) (int, bool) {
	h := int(adler32.Checksum([]byte(str))) % nodeNum
	if h == nodeIdx {
		return h, true
	}
	return h, false
}

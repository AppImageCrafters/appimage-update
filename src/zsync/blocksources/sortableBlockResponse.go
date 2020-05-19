package blocksources

import (
	"appimage-update/src/zsync/patcher"
)

type PendingResponses []patcher.BlockReponse

func (r PendingResponses) Len() int {
	return len(r)
}

func (r PendingResponses) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r PendingResponses) Less(i, j int) bool {
	return r[i].StartBlock < r[j].StartBlock
}

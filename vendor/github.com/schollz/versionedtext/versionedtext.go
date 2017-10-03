package versionedtext

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// VersionedText is the main container for the diff functions
type VersionedText struct {
	CurrentText string
	Diffs       map[int64]string
}

// NewVersionedText returns a new VersionedText object
func NewVersionedText(text string) VersionedText {
	data := VersionedText{CurrentText: "", Diffs: make(map[int64]string)}
	data.Update(text)
	return data
}

// diffRebuildtexts will use the patches to rebuild the text
func (vt *VersionedText) diffRebuildtexts(Diffs []diffmatchpatch.Diff) []string {
	text := []string{"", ""}
	for _, diff := range Diffs {
		if diff.Type != diffmatchpatch.DiffInsert {
			text[0] += diff.Text
		}
		if diff.Type != diffmatchpatch.DiffDelete {
			text[1] += diff.Text
		}
	}
	return text
}

// rebuildTextsToDiffN will use patches to build the text to a certain point
func (vt *VersionedText) rebuildTextsToDiffN(timestamp int64, snapshots []int64) (string, error) {
	dmp := diffmatchpatch.New()
	lastText := ""

	for _, snapshot := range snapshots {

		diff := vt.Diffs[snapshot]
		seq1, _ := dmp.DiffFromDelta(lastText, diff)
		textsLinemode := vt.diffRebuildtexts(seq1)
		rebuilt := textsLinemode[len(textsLinemode)-1]

		if snapshot == timestamp {
			return rebuilt, nil
		}
		lastText = rebuilt
	}

	return "", fmt.Errorf("Could not rebuild from Diffs")
}

// GetCurrent returns the latest version
func (vt *VersionedText) GetCurrent() string {
	return vt.CurrentText
}

// NumEdits returns the number of edits
func (vt *VersionedText) NumEdits() int {
	return len(vt.Diffs)
}

// LastEditTime returns the last time it was edited
func (vt *VersionedText) LastEditTime() int64 {
	max := int64(0)
	for key := range vt.Diffs {
		if key > max {
			max = key
		}
	}
	return max
}

// Update adds a new version to the current versions
func (vt *VersionedText) Update(newText string) {
	// check for changes
	if vt.CurrentText == newText {
		return
	}

	dmp := diffmatchpatch.New()
	delta := dmp.DiffToDelta(dmp.DiffMain(vt.CurrentText, newText, true))
	vt.CurrentText = newText
	now := time.Now().UnixNano()
	vt.Diffs[now] = delta
}

// GetSnapshots returns a sorted list of integers which
// represent timestamps for each snapshot
func (vt *VersionedText) GetSnapshots() []int64 {
	keys := make([]int64, 0, len(vt.Diffs))
	for k := range vt.Diffs {
		keys = append(keys, k)
	}
	// SORT KEYS
	keys = mergeSortInt64(keys)
	return keys
}

// GetMajorSnapshots returns a sorted list of integers which
// represent timestamps for each snapshot, where snapshots
// must be at least the number of specified seconds apart.
func (vt *VersionedText) GetMajorSnapshots(seconds int64) []int64 {
	keys := vt.GetSnapshots()
	newKeys := make([]int64, len(keys))
	newKeysI := 0
	for i, key := range keys {
		if i == 0 && len(keys) > 1 {
			continue
		}
		if i == len(keys)-1 {
			newKeys[newKeysI] = key
			newKeysI++
			continue
		}
		if key-keys[i-1] > seconds*1000000000 {
			newKeys[newKeysI] = key
			newKeysI++
		}
	}
	return newKeys[0:newKeysI]
}

// GetMajorSnapshotsAndChangeSums returns the major snapshots timestamps and the changesums
// as two arrays
func (vt *VersionedText) GetMajorSnapshotsAndChangeSums(seconds int64) ([]int64, []int) {
	keys := vt.GetSnapshots()
	changeSums := vt.GetChangeSums()

	majorChangeSums := make([]int, len(keys))
	newKeys := make([]int64, len(keys))
	newKeysI := 0
	cumulativeSum := 0
	for i, key := range keys {
		cumulativeSum = cumulativeSum + changeSums[i]
		if i == 0 && len(keys) > 1 {
			continue
		}
		if i == len(keys)-1 {
			newKeys[newKeysI] = key
			majorChangeSums[newKeysI] = cumulativeSum
			cumulativeSum = 0
			newKeysI++
			continue
		}
		if key-keys[i-1] > seconds*1000000000 {
			majorChangeSums[newKeysI] = cumulativeSum
			newKeys[newKeysI] = key
			cumulativeSum = 0
			newKeysI++
		}
	}
	return newKeys[0:newKeysI], majorChangeSums[0:newKeysI]
}

// GetChangeSums returns the length of each change
func (vt *VersionedText) GetChangeSums() []int {
	keys := vt.GetSnapshots()
	changeSums := make([]int, len(keys))
	for i, key := range keys {
		changeSums[i] = 0
		for _, val := range strings.Split(vt.Diffs[key], "\t") {
			if val[0:1] == "-" {
				a, _ := strconv.Atoi(val[1:])
				changeSums[i] = changeSums[i] - a
			} else if val[0:1] == "+" {
				changeSums[i] = changeSums[i] + len(val[1:])
			}
		}
	}
	return changeSums
}

// GetPreviousByTimestamp uses the diffs to rebuild to the specified snapshot
func (vt *VersionedText) GetPreviousByTimestamp(timestamp int64) (string, error) {

	// check inputs
	if 0 > timestamp {
		return "", fmt.Errorf("Timestamps most be positive integer")
	}

	// get change snapshot
	snapshots := vt.GetSnapshots()

	// default to first value
	ts := snapshots[0]

	// find timestamp
	for _, snapshot := range snapshots {
		if timestamp >= snapshot && ts < snapshot {
			ts = snapshot
		}
	}

	// use timestamp to find value
	oldValue, err := vt.rebuildTextsToDiffN(ts, snapshots)
	return oldValue, err
}

// GetPreviousByIndex returns a snapshot based on the index
func (vt *VersionedText) GetPreviousByIndex(idx int) (string, error) {
	if vt.NumEdits() == 0 {
		return vt.GetCurrent(), nil
	}
	// check inputs
	if 0 > idx {
		return "", fmt.Errorf("Index most be positive integer")
	}

	// get change snapshots
	snapshots := vt.GetSnapshots()

	// if index greater than length of snapshot
	// default to last snapshot
	if idx > len(snapshots)-1 {
		idx = len(snapshots) - 1
	}

	// use index to find timestamp
	ts := snapshots[idx]

	// use timestamp to find value
	oldValue, err := vt.rebuildTextsToDiffN(ts, snapshots)
	return oldValue, err
}

func mergeInt64(l, r []int64) []int64 {
	ret := make([]int64, 0, len(l)+len(r))
	for len(l) > 0 || len(r) > 0 {
		if len(l) == 0 {
			return append(ret, r...)
		}
		if len(r) == 0 {
			return append(ret, l...)
		}
		if l[0] <= r[0] {
			ret = append(ret, l[0])
			l = l[1:]
		} else {
			ret = append(ret, r[0])
			r = r[1:]
		}
	}
	return ret
}

func mergeSortInt64(s []int64) []int64 {
	if len(s) <= 1 {
		return s
	}
	n := len(s) / 2
	l := mergeSortInt64(s[:n])
	r := mergeSortInt64(s[n:])
	return mergeInt64(l, r)
}

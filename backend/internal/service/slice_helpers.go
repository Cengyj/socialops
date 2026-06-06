package service

// UniqueInt64sPreserveOrder removes duplicate IDs without changing the first-seen
// order. Callers still own validation; this helper must not silently drop
// malformed IDs that should be rejected by the request contract.
func UniqueInt64sPreserveOrder(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

package cache

func AsMap[T any](xs []StorageItemMulti[T]) map[string]T {
	m := make(map[string]T, len(xs))
	for _, v := range xs {
		m[v.Key] = v.Value
	}

	return m
}

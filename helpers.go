package main

func countBy[T any, K comparable](s []T, key func(T) K) map[K]int {
	counts := make(map[K]int)
	for _, v := range s {
		counts[key(v)]++
	}
	return counts
}

package main

import "iter"

func dedupedBy[T any, K comparable](s []T, key func(T) K) iter.Seq[T] {
	return func(yield func(T) bool) {
		seen := make(set[K])
		for _, v := range s {
			k := key(v)
			if seen.has(k) {
				continue
			}
			seen.add(k)
			if !yield(v) {
				return
			}
		}
	}
}

func countBy[T any, K comparable](s []T, key func(T) K) map[K]int {
	counts := make(map[K]int)
	for _, v := range s {
		counts[key(v)]++
	}
	return counts
}

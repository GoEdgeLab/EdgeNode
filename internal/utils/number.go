package utils

func MinInt(min1 int, min2 int) int {
	if min1 < min2 {
		return min1
	}
	return min2
}

func MaxInt(min1 int, min2 int) int {
	if min1 < min2 {
		return min2
	}
	return min1
}

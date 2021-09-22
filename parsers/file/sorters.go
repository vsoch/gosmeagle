package file

// SortByAddress
type SortByAddress []Symbol

func (x SortByAddress) Less(i, j int) bool {
	return x[i].Address < x[j].Address
}

// Length returns the number of symbols in the sorter
func (x SortByAddress) Len() int {
	return len(x)
}

// Swap swips two symbols
func (x SortByAddress) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

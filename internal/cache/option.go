package cache

type OptionInterface interface {
}

type PiecesOption struct {
	Count int
}

func NewPiecesOption(count int) *PiecesOption {
	return &PiecesOption{Count: count}
}

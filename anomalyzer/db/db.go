package db

type DbClient interface {
	Get(timestamp int64) ([]float64, error)
	Update([]float64) error
	GetAndUpdate(timestamp int64) error
	Eval() float64
}

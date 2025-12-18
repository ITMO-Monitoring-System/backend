package domain

type Department struct {
	ID    int64
	Code  string
	Name  string
	Alias *string
}

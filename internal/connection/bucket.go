package connection

type Bucket struct {
	Name string
}

func NewBucket(name string) *Bucket {
	return &Bucket{
		Name: name,
	}
}

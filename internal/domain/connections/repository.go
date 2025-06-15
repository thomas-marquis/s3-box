package connections

type Repository interface {
	Get() (*Connections, error)
	Save(conn *Connections) error
}

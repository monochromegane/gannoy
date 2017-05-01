package gannoy

type Storage interface {
	Create(Node) (int, error)
	Find(int) (Node, error)
	Update(Node) error
	UpdateParent(int, int, int) error
	Delete(Node) error
	Iterate(chan Node)
}

package gannoy

type Queue struct {
	priority float64
	value    int
}

func (q *Queue) Less(other interface{}) bool {
	return q.priority > other.(*Queue).priority
}

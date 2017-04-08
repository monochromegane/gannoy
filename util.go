package annoy

import (
	"math"
)

func twoMeans(distance Distance, nodes []Node, f int, random Random, cosine bool) ([]float64, []float64) {
	iteration_steps := 200
	count := len(nodes)

	i := random.index(count)
	j := random.index(count - 1)
	if j >= i {
		j++
	}
	iv := make([]float64, len(nodes[i].v))
	copy(iv, nodes[i].v)

	jv := make([]float64, len(nodes[j].v))
	copy(jv, nodes[j].v)

	if cosine {
		normalize(iv, f)
		normalize(jv, f)
	}

	ic := 1
	jc := 1

	for l := 0; l < iteration_steps; l++ {
		k := random.index(count)

		di := float64(ic) * distance.distance(iv, nodes[k].v, f)
		dj := float64(jc) * distance.distance(jv, nodes[k].v, f)

		norm := 1.0
		if cosine {
			norm = getNorm(nodes[k].v, f)
		}

		if di < dj {
			for z := 0; z < f; z++ {
				iv[z] = (iv[z]*float64(ic) + nodes[k].v[z]/norm) / float64(ic+1)
			}
			ic++
		} else if dj < di {
			for z := 0; z < f; z++ {
				jv[z] = (jv[z]*float64(jc) + nodes[k].v[z]/norm) / float64(jc+1)
			}
			jc++
		}
	}
	return iv, jv
}

func normalize(v []float64, f int) []float64 {
	norm := getNorm(v, f)
	for z := 0; z < f; z++ {
		v[z] /= norm
	}
	return v
}

func getNorm(v []float64, f int) float64 {
	var sq_norm float64
	for z := 0; z < f; z++ {
		sq_norm += v[z] * v[z]
	}
	return math.Sqrt(sq_norm)
}

package mt

// A Vec is a 3D vector in units of 0.1 nodes.
type Vec [3]float32

// Add returns v+w.
func (v Vec) Add(w Vec) Vec {
	for i := range v {
		v[i] += w[i]
	}
	return v
}

// Sub returns v-w.
func (v Vec) Sub(w Vec) Vec {
	for i := range v {
		v[i] -= w[i]
	}
	return v
}

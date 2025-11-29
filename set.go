package idl

func makeSet[t comparable]() *set[t] {
	return &set[t]{
		v: make(map[t]struct{}),
	}
}

type set[t comparable] struct {
	v map[t]struct{}
}

func (s *set[t]) add(v t) {
	s.v[v] = struct{}{}
}

func (s *set[t]) has(v t) bool {
	_, ok := s.v[v]
	return ok
}

func (s *set[t]) values() []t {
	vs := make([]t, 0, len(s.v))
	for v := range s.v {
		vs = append(vs, v)
	}
	return vs
}

func (s *set[t]) remove(v t) {
	delete(s.v, v)
}

func (s *set[t]) clear() {
	s.v = make(map[t]struct{})
}

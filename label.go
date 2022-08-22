package ipam

type LabelMap map[string]string

func (lm LabelMap) Copy() LabelMap {
	m := make(LabelMap)
	for k, v := range lm {
		m[k] = v
	}
	return m
}

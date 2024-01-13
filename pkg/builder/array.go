package builder

type arrayField struct {
	values []Jsonable
}

var (
	_ Jsonable = &arrayField{}
)

func Array(items []Jsonable) *arrayField {
	return &arrayField{
		values: items,
	}
}

func (s *arrayField) IsEmpty() bool {
	if len(s.values) == 0 {
		return true
	}

	for _, v := range s.values {
		if !v.IsEmpty() {
			return false
		}
	}

	return true
}

func (s *arrayField) ToJson() string {
	str := "["

	for i, item := range s.values {
		if item == nil {
			str += Null().ToJson()
		} else {
			str += item.ToJson()
		}

		if i != len(s.values)-1 {
			str += ","
		}
	}

	return str + "]"
}

func (s *arrayField) Raw() interface{} {
	res := make([]interface{}, len(s.values))

	for i, item := range s.values {
		if item == nil {
			res[i] = Null().Raw()
		} else {
			res[i] = item.Raw()
		}
	}

	return res
}
package rabbitgo

import "reflect"

func inSlice(v interface{}, s interface{}) bool {
	val := reflect.Indirect(reflect.ValueOf(v))
	if val.Kind() == reflect.Slice {
		return false
	}

	sv := reflect.Indirect(reflect.ValueOf(s))
	if sv.Kind() != reflect.Slice {
		return false
	}

	st := reflect.TypeOf(s).Elem().String()
	vt := reflect.TypeOf(v).String()
	if st != vt {
		return false
	}

	switch vt {
	case "string":
		for _, vv := range s.([]string) {
			if vv == v {
				return true
			}
		}
	case "int":
		for _, vv := range s.([]int) {
			if vv == v {
				return true
			}
		}
	case "int64":
		for _, vv := range s.([]int64) {
			if vv == v {
				return true
			}
		}
	case "float64":
		for _, vv := range s.([]float64) {
			if vv == v {
				return true
			}
		}
	default:
		return false
	}

	return false
}

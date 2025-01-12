package iter

func MapErr[F any, T any](xs []F, fun func(F) (T, error)) ([]T, error) {
	result := make([]T, len(xs))

	for i, x := range xs {
		var err error

		result[i], err = fun(x)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

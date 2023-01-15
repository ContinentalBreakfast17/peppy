package common

type Object[v any] map[string]v
type ObjectEntry[v any] struct {
	Key   string
	Value v
}
type List[v any] []v
type ObjectEntries[v any] []ObjectEntry[v]

func (object Object[v]) Entries() List[ObjectEntry[v]] {
	result := List[ObjectEntry[v]]{}
	for key, value := range object {
		result = append(result, ObjectEntry[v]{
			Key:   key,
			Value: value,
		})
	}
	return result
}

func (object Object[v]) Values() List[v] {
	result := List[v]{}
	for _, value := range object {
		result = append(result, value)
	}
	return result
}

func (object Object[v]) Keys() []string {
	result := []string{}
	for key := range object {
		result = append(result, key)
	}
	return result
}

func (entries ObjectEntries[v]) ToObject() Object[v] {
	result := Object[v]{}
	for _, entry := range entries {
		result[entry.Key] = entry.Value
	}
	return result
}

func TransformList[before any, after any](list List[before], transform func(item before) after) List[after] {
	result := make(List[after], len(list))
	for i, item := range list {
		result[i] = transform(item)
	}
	return result
}

func TransformMapValues[before any, after any](m map[string]before, transform func(item before) after) map[string]after {
	return ObjectEntries[after](TransformList[ObjectEntry[before], ObjectEntry[after]](
		Object[before](m).Entries(),
		func(entry ObjectEntry[before]) ObjectEntry[after] {
			return ObjectEntry[after]{
				Key: entry.Key, Value: transform(entry.Value),
			}
		},
	)).ToObject()
}
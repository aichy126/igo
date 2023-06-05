package util

// 查找数组中某值是否在数组中
func ArrayNotInArrayString(original []string, search []string) []string {
	var result []string
	originalMap := make(map[string]bool)
	for _, v := range original {
		originalMap[v] = true
	}
	for _, v := range search {
		if _, ok := originalMap[v]; !ok {
			result = append(result, v)
		}
	}
	return result
}

// 查找某值是否在数组中
func InArrayString(v string, m *[]string) bool {
	for _, value := range *m {
		if value == v {
			return true
		}
	}
	return false
}

package merge

func isYAMLTable(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}

// MergeMaps merges any number of maps together, with maps later in the slice taking
// precedent
func MergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	// merge bottom-up
	switch {
	case len(maps) > 2:
		mLen := len(maps)
		newMaps := maps[0 : mLen-2]
		newMaps = append(newMaps, MergeMaps(maps[mLen-2], maps[mLen-1]))
		return MergeMaps(newMaps...)
	case len(maps) == 2:
		if maps[0] == nil {
			return maps[1]
		}
		if maps[1] == nil {
			return maps[0]
		}
		for key, map0Val := range maps[0] {
			if map1Val, ok := maps[1][key]; ok && map1Val == nil {
				delete(maps[1], key)
			} else if !ok {
				maps[1][key] = map0Val
			} else if isYAMLTable(map0Val) {
				if isYAMLTable(map1Val) {
					MergeMaps(map0Val.(map[string]interface{}), map1Val.(map[string]interface{}))
				}
			}
		}
		return maps[1]
	case len(maps) == 1:
		return maps[0]
	}

	return nil
}

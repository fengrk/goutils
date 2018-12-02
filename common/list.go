package common

import "time"

func RemoveDuplicateContent(rawList []string) (newList []string) {
	hashSet := map[string]bool{}
	
	for _, content := range rawList {
		_, exists := hashSet[content]
		if !exists {
			newList = append(newList, content)
			hashSet[content] = true
		}
	}
	return newList
	
}

func GetMinDuration(duration ... time.Duration) time.Duration {
	minDuration := duration[0]
	for _, d := range duration {
		if d <= minDuration {
			minDuration = d
		}
	}
	return minDuration
}

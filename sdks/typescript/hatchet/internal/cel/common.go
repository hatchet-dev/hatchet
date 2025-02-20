package cel

import "fmt"

func Str(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

func Int(i int) string {
	return fmt.Sprintf("%d", i)
}

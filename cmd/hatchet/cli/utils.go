package cli

import "os"

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false, nil
	}

	return err == nil, err
}

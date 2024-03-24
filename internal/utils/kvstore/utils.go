// Copyright 2024 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .

package kvstore

import (
	"errors"
	"os"
	"regexp"
	"strings"
)

var nameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

// IsValidName check if store name or database name or table name is valid
func IsValidName(name string) bool {
	return nameRegexp.MatchString(name)
}

// RemoveStore remove store directory
func RemoveStore(path string) error {
	var errNotStoreDirectory = errors.New("not store directory")

	if strings.HasSuffix(path, StoreSuffix) {
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// validate store
		{
			_, err = os.Stat(path + "/CURRENT")
			if err != nil {
				return errNotStoreDirectory
			}
		}
		{
			_, err = os.Stat(path + "/LOCK")
			if err != nil {
				return errNotStoreDirectory
			}
		}

		return os.RemoveAll(path)
	}

	return errNotStoreDirectory
}

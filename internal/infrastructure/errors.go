package infrastructure

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

const mysqlDuplicateEntryCode = 1062

func IsDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == mysqlDuplicateEntryCode
	}

	return false
}

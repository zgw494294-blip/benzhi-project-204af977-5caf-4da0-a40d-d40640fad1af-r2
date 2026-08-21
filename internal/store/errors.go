package store

import "errors"

var (
	ErrCorruptLedger = errors.New("账本文件损坏")
	ErrSchemaVersion = errors.New("账本 schemaVersion 不受支持")
)

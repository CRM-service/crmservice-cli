package api

import "errors"

var (
	ErrEmptyRecord         = errors.New("empty record")
	ErrBulkUpdateMissingID = errors.New("bulk-update requires id in each record")
)

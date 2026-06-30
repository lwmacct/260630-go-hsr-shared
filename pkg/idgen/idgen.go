package idgen

import "github.com/google/uuid"

func NewUUID7() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic("idgen.NewUUID7: " + err.Error())
	}
	return id.String()
}

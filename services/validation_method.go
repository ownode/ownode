package services

import (
	"strings"
	// validator "github.com/asaskevich/govalidator"
)

type CustomValidator struct {

}

// check if a string is null or empty
func (cv *CustomValidator) IsEmpty(str string) bool {
	if strings.TrimSpace(str) == "" {
		return true
	}
	return false
}
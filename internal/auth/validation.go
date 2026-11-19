package auth

import "example.com/pkg/validation"

func validLoginInput(email, password string) bool {
	return validation.ValidEmail(email) && len(password) > 0 && len(password) <= 72
}

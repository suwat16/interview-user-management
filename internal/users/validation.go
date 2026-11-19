package users

import (
	"strings"

	"example.com/pkg/validation"
)

type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}

type CreateInput struct {
	Name     string
	Email    string
	Password string
}

func (input CreateInput) Validate() (CreateInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	if input.Name == "" || !validation.ValidEmail(input.Email) || input.Password == "" {
		return CreateInput{}, ValidationError("name, valid email, and password are required")
	}
	if len(input.Password) > 72 {
		return CreateInput{}, ValidationError("password must be at most 72 bytes")
	}
	return input, nil
}

func (fields UpdateFields) Validate() (UpdateFields, error) {
	if fields.Name == nil && fields.Email == nil {
		return UpdateFields{}, ValidationError("name or email is required")
	}
	if fields.Name != nil {
		name := strings.TrimSpace(*fields.Name)
		if name == "" {
			return UpdateFields{}, ValidationError("name cannot be blank")
		}
		fields.Name = &name
	}
	if fields.Email != nil {
		email := strings.TrimSpace(*fields.Email)
		if !validation.ValidEmail(email) {
			return UpdateFields{}, ValidationError("invalid email")
		}
		fields.Email = &email
	}
	return fields, nil
}

// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
	"time"
)

type EditRole struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type EditUser struct {
	Email string      `json:"email"`
	Roles []*EditRole `json:"roles"`
}

type EnrichUser struct {
	Email    string `json:"email"`
	FullName string `json:"fullName"`
	Profile  string `json:"profile"`
}

type Role struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type User struct {
	Email       string     `json:"email"`
	Active      bool       `json:"active"`
	Roles       []*Role    `json:"roles"`
	FullName    *string    `json:"fullName"`
	Profile     *string    `json:"profile"`
	FirstAccess *time.Time `json:"firstAccess"`
	LastAccess  *time.Time `json:"lastAccess"`
}

type AdminEventType string

const (
	AdminEventTypeEditUser AdminEventType = "EDIT_USER"
)

var AllAdminEventType = []AdminEventType{
	AdminEventTypeEditUser,
}

func (e AdminEventType) IsValid() bool {
	switch e {
	case AdminEventTypeEditUser:
		return true
	}
	return false
}

func (e AdminEventType) String() string {
	return string(e)
}

func (e *AdminEventType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = AdminEventType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid AdminEventType", str)
	}
	return nil
}

func (e AdminEventType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

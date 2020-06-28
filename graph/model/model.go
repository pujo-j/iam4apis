package model

import (
	"strings"
	"time"
)

type AdminEvent struct {
	ID      string    `json:"id"`
	Ts      time.Time `json:"ts"`
	AdminID string    `json:"admin"`
	UserID  string    `json:"user"`
	Roles   []*Role   `json:"roles"`
}

func (u *User) IsInRole(role string, path string) bool {
	for _, r := range u.Roles {
		if strings.HasPrefix(path, r.Path) && r.Name == role {
			return true
		}
	}
	return false
}

func (u *User) Edit() *EditUser {
	roles := make([]*EditRole, 0, len(u.Roles))
	for _, role := range u.Roles {
		roles = append(roles, &EditRole{
			Name: role.Name,
			Path: role.Path,
		})
	}
	return &EditUser{
		Email: u.Email,
		Roles: roles,
	}
}

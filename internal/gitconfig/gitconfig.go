package gitconfig

import (
	"os/exec"
	"strings"
)

// Identity holds git user.name and user.email.
type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Current reads git config user.email and user.name.
func Current() Identity {
	return Identity{
		Name:  read("user.name"),
		Email: read("user.email"),
	}
}

func read(key string) string {
	out, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

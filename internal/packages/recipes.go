package packages

import (
	"embed"
	"strings"

	"davish/internal/shellx"
)

//go:embed recipes/*.sh
var recipeFS embed.FS

// Recipe installs one program with the server's package manager.
type Recipe struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// Apps are the ids of App that show it is installed.
	Apps []string `json:"apps"`
	// Script is the whole script, shown before it runs.
	Script string `json:"script"`
	// Command runs Script, typed into a terminal.
	Command string `json:"command"`
}

var recipeList = []struct {
	id, label string
	apps      []string
}{
	{"nginx", "nginx", []string{"nginx"}},
	{"docker", "Docker Engine", []string{"docker"}},
	{"node", "Node.js (nvm)", []string{"node"}},
	{"php", "PHP-FPM", []string{"php"}},
	{"mariadb", "MariaDB", []string{"mariadb", "mysql"}},
	{"postgres", "PostgreSQL", []string{"postgres"}},
	{"redis", "Redis", []string{"redis"}},
	{"certbot", "Certbot", []string{"certbot"}},
	{"fail2ban", "fail2ban", []string{"fail2ban"}},
	{"ufw", "ufw", []string{"ufw"}},
}

// Recipes returns every install recipe, each script with the shared
// prelude that finds apt, dnf or apk.
func Recipes() []Recipe {
	common, _ := recipeFS.ReadFile("recipes/_common.sh")
	out := make([]Recipe, 0, len(recipeList))
	for _, r := range recipeList {
		body, err := recipeFS.ReadFile("recipes/" + r.id + ".sh")
		if err != nil {
			continue
		}
		script := string(common) + "\n" + strings.TrimRight(string(body), "\n") + "\n"
		out = append(out, Recipe{
			ID: r.id, Label: r.label, Apps: r.apps, Script: script,
			Command: shellx.Cmd("sh", "-c", script).String(),
		})
	}
	return out
}

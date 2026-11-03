package git

import (
	"slices"
	"testing"
)

func TestRecipeFor(t *testing.T) {
	for _, c := range []struct {
		name  string
		files []string
		want  []string
	}{
		{"nothing", nil, []string{"git pull --ff-only", "# sudo systemctl restart my-app"}},
		{"npm build", []string{"package.json", "package-lock.json", buildScript},
			[]string{"git pull --ff-only", "npm ci", "npm run build", "# sudo systemctl restart my-app"}},
		{"pnpm build", []string{"package.json", "pnpm-lock.yaml", buildScript},
			[]string{"git pull --ff-only", "pnpm install --frozen-lockfile", "pnpm run build", "# sudo systemctl restart my-app"}},
		{"yarn, no build", []string{"package.json", "yarn.lock"},
			[]string{"git pull --ff-only", "yarn install --frozen-lockfile", "# sudo systemctl restart my-app"}},
		{"no lockfile still builds", []string{"package.json", buildScript},
			[]string{"git pull --ff-only", "npm run build", "# sudo systemctl restart my-app"}},
		{"compose wins", []string{"package.json", "package-lock.json", buildScript, "Dockerfile", "compose.yaml"},
			[]string{"git pull --ff-only", "docker compose up -d --build"}},
		{"go", []string{"go.mod", "Makefile"},
			[]string{"git pull --ff-only", "go build -o bin/ ./...", "# sudo systemctl restart my-app"}},
		{"make", []string{"Makefile"},
			[]string{"git pull --ff-only", "make", "# sudo systemctl restart my-app"}},
		{"dockerfile", []string{"Dockerfile"},
			[]string{"git pull --ff-only", "# docker build -t my-app . && docker restart my-app", "# sudo systemctl restart my-app"}},
	} {
		if got := recipeFor(c.files); !slices.Equal(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

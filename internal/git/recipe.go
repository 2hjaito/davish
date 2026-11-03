package git

import "slices"

// recipeFor guesses a deploy recipe from the files at the root, as
// statusLine lists them. Compose builds and runs everything in its
// containers, so it replaces the steps on the server itself. Steps it
// cannot be sure of are left as # hints.
func recipeFor(files []string) []string {
	has := func(f string) bool { return slices.Contains(files, f) }
	steps := []string{"git pull --ff-only"}
	for _, f := range composeFiles {
		if has(f) {
			return append(steps, "docker compose up -d --build")
		}
	}
	install := installFor(files)
	if install != "" {
		steps = append(steps, install)
	}
	switch {
	case has("package.json") && has(buildScript):
		steps = append(steps, buildFor(install))
	case has("go.mod"):
		steps = append(steps, "go build -o bin/ ./...")
	case has("Makefile"):
		steps = append(steps, "make")
	}
	if has("Dockerfile") {
		steps = append(steps, "# docker build -t my-app . && docker restart my-app")
	}
	return append(steps, "# sudo systemctl restart my-app")
}

// buildFor runs the build script with the package manager that installed.
func buildFor(install string) string {
	switch install {
	case "pnpm install --frozen-lockfile":
		return "pnpm run build"
	case "yarn install --frozen-lockfile":
		return "yarn build"
	}
	return "npm run build"
}

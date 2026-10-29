package packages

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRecipes(t *testing.T) {
	rs := Recipes()
	if len(rs) != len(recipeList) {
		t.Fatalf("%d recipes, want %d", len(rs), len(recipeList))
	}
	for _, r := range rs {
		if !strings.HasPrefix(r.Script, "# Davish install recipe") || !strings.HasPrefix(r.Command, "sh -c '") {
			t.Errorf("%s: %q", r.ID, r.Command[:40])
		}
		// Every script must at least parse.
		cmd := exec.Command("sh", "-n")
		cmd.Stdin = strings.NewReader(r.Script)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s: %v %s", r.ID, err, out)
		}
	}
}

package packages

import (
	"context"
	"testing"

	"davish/internal/audit"
)

func TestPlanSetup(t *testing.T) {
	tests := []struct {
		m, tool string
		want    string
	}{
		{Apt, ToolDocker, `sudo apt-get update && sudo apt-get install docker.io && sudo systemctl enable --now docker && sudo usermod -aG docker "$USER"`},
		{Dnf, ToolDocker, `sudo dnf install moby-engine && sudo systemctl enable --now docker && sudo usermod -aG docker "$USER"`},
		{Yum, ToolDocker, `sudo yum install docker && sudo systemctl enable --now docker && sudo usermod -aG docker "$USER"`},
		{Apk, ToolDocker, `sudo apk update && sudo apk add docker && sudo rc-update add docker && sudo rc-service docker start && sudo addgroup "$USER" docker`},

		{Apt, ToolNginx, "sudo apt-get update && sudo apt-get install nginx && sudo systemctl enable --now nginx"},
		{Dnf, ToolNginx, "sudo dnf install nginx && sudo systemctl enable --now nginx"},
		{Yum, ToolNginx, "sudo yum install nginx && sudo systemctl enable --now nginx"},
		{Apk, ToolNginx, "sudo apk update && sudo apk add nginx && sudo rc-update add nginx && sudo rc-service nginx start"},

		{Apt, ToolGit, "sudo apt-get update && sudo apt-get install git"},
		{Dnf, ToolGit, "sudo dnf install git"},
		{Yum, ToolGit, "sudo yum install git"},
		{Apk, ToolGit, "sudo apk update && sudo apk add git"},

		{Apt, ToolMariaDB, "sudo apt-get update && sudo apt-get install mariadb-server && sudo systemctl enable --now mariadb"},
		{Dnf, ToolMariaDB, "sudo dnf install mariadb-server && sudo systemctl enable --now mariadb"},
		{Yum, ToolMariaDB, "sudo yum install mariadb-server && sudo systemctl enable --now mariadb"},
		{Apk, ToolMariaDB, "sudo apk update && sudo apk add mariadb mariadb-client && sudo rc-service mariadb setup && sudo rc-update add mariadb && sudo rc-service mariadb start"},

		{Apt, ToolPostgres, "sudo apt-get update && sudo apt-get install postgresql && sudo systemctl enable --now postgresql"},
		{Dnf, ToolPostgres, "sudo dnf install postgresql-server && sudo postgresql-setup --initdb && sudo systemctl enable --now postgresql"},
		{Yum, ToolPostgres, "sudo yum install postgresql-server && sudo postgresql-setup --initdb && sudo systemctl enable --now postgresql"},
		{Apk, ToolPostgres, "sudo apk update && sudo apk add postgresql && sudo rc-update add postgresql && sudo rc-service postgresql start"},
	}
	for _, tt := range tests {
		got, err := PlanSetup(tt.m, tt.tool)
		if err != nil || got.Command != tt.want {
			t.Errorf("%s %s = %q, %v; want %q", tt.m, tt.tool, got.Command, err, tt.want)
		}
		if got.Tool != tt.tool || got.Manager != tt.m || got.Relogin != (tt.tool == ToolDocker) {
			t.Errorf("%s %s = %+v", tt.m, tt.tool, got)
		}
	}
}

func TestPlanSetupUnknown(t *testing.T) {
	got, err := PlanSetup("pacman", ToolNginx)
	if err != nil || got.Command != "" || got.Manager != "pacman" {
		t.Errorf("unknown manager = %+v, %v", got, err)
	}
	got, err = PlanSetup("", ToolDocker)
	if err != nil || got.Command != "" || got.Relogin {
		t.Errorf("no manager = %+v, %v", got, err)
	}
	if got, err := PlanSetup(Apt, "rm -rf /"); err == nil {
		t.Errorf("unknown tool planned %+v", got)
	}
}

type fakeRunner struct {
	out   string
	calls int
}

func (f *fakeRunner) RunLine(context.Context, string, string, string) (audit.Result, error) {
	f.calls++
	return audit.Result{Stdout: []byte(f.out)}, nil
}

func TestServiceSetup(t *testing.T) {
	run := &fakeRunner{out: "dnf\n"}
	s := New(run, nil)
	got, err := s.Setup(context.Background(), "srv", ToolGit)
	if err != nil || got.Command != "sudo dnf install git" {
		t.Errorf("setup = %+v, %v", got, err)
	}
	if _, err := s.Setup(context.Background(), "srv", "nope"); err == nil || run.calls != 1 {
		t.Errorf("unknown tool: err %v, %d calls", err, run.calls)
	}
}

package git

import (
	"os"
	"testing"
)

func TestParseDeployInfo(t *testing.T) {
	out, err := os.ReadFile("testdata/deployinfo.txt")
	if err != nil {
		t.Fatal(err)
	}
	info := parseDeployInfo("/srv/shop", sections(string(out)))

	if info.Pull.Command != "git pull --ff-only" {
		t.Errorf("pull = %+v", info.Pull)
	}
	if info.Install == nil || info.Install.Command != "pnpm install --frozen-lockfile" {
		t.Errorf("install = %+v", info.Install)
	}
	var builds []string
	for _, b := range info.Builds {
		builds = append(builds, b.Name+"="+b.Command)
	}
	want := []string{"build=pnpm run build", "lint=pnpm run lint", "start:prod=pnpm run start:prod", "make=make"}
	if len(builds) != len(want) {
		t.Fatalf("builds = %q", builds)
	}
	for i := range want {
		if builds[i] != want[i] {
			t.Errorf("build %d = %q, want %q", i, builds[i], want[i])
		}
	}

	var runners []string
	for _, r := range info.Runners {
		runners = append(runners, r.Kind+":"+r.Name+"="+r.Command)
	}
	wantRunners := []string{
		"pm2:shop=pm2 restart shop",
		"pm2:shop worker=pm2 restart 'shop worker'",
		"systemd:shop-api.service=sudo systemctl restart shop-api.service",
		"systemd:shop-cron.service=sudo systemctl restart shop-cron.service",
		"compose:shop=docker compose up -d --build",
	}
	if len(runners) != len(wantRunners) {
		t.Fatalf("runners = %q", runners)
	}
	for i := range wantRunners {
		if runners[i] != wantRunners[i] {
			t.Errorf("runner %d = %q, want %q", i, runners[i], wantRunners[i])
		}
	}
	if !info.Runners[2].Sudo || info.Runners[0].Sudo || !info.Compose {
		t.Errorf("sudo/compose flags wrong: %+v", info)
	}
}

func TestParseDeployInfoAsRootAndEmpty(t *testing.T) {
	info := parseDeployInfo("/srv/a", sections("@@uid\n0\n@@root\n@@package\n\n@@pm2\nnot json\n@@systemd\n"+
		"Id=a.service\nWorkingDirectory=/srv/a\nExecStart=\n\n@@compose\n"))
	if info.Install != nil || len(info.Builds) != 0 || info.Compose {
		t.Errorf("info = %+v", info)
	}
	if len(info.Runners) != 1 || info.Runners[0].Command != "systemctl restart a.service" || info.Runners[0].Sudo {
		t.Errorf("runners = %+v", info.Runners)
	}
}

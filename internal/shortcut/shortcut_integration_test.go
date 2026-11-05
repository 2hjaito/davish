//go:build integration

package shortcut_test

import (
	"context"
	"errors"
	"testing"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/fs"
	"davish/internal/shortcut"
	"davish/internal/testsshd"
)

type discard struct{}

func (discard) Emit(string, any) {}

func TestScriptChangeIsRefused(t *testing.T) {
	m, id := testsshd.Connect(t)
	log := audit.New(discard{})
	files := fs.New(m, audit.NewRunner(m, log), testsshd.Quiet())
	s := shortcut.New(files, log)
	ctx := context.Background()
	testsshd.Run(t, m, id, `set -e; rm -rf ~/shortcuts; mkdir -p ~/shortcuts/app
printf '#!/bin/sh\necho one\n' > ~/shortcuts/deploy.sh; chmod +x ~/shortcuts/deploy.sh
echo note > ~/shortcuts/notes.txt; ln -s app ~/shortcuts/current`)
	t.Cleanup(func() { testsshd.Run(t, m, id, `rm -rf ~/shortcuts`) })

	const dir = "/home/tester/shortcuts/"
	for p, want := range map[string]string{
		"app": shortcut.KindFolder, "current": shortcut.KindFolder,
		"notes.txt": shortcut.KindFile, "deploy.sh": shortcut.KindScript,
	} {
		got, err := s.Inspect(ctx, id, dir+p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Kind != want {
			t.Errorf("%s: kind = %s, want %s", p, got.Kind, want)
		}
	}

	seen, err := s.Inspect(ctx, id, dir+"deploy.sh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Run(ctx, id, dir+"deploy.sh", seen.Hash); err != nil {
		t.Fatal(err)
	}
	testsshd.Run(t, m, id, `echo 'echo two' >> ~/shortcuts/deploy.sh`)
	var ae *apperr.Error
	if _, err := s.Run(ctx, id, dir+"deploy.sh", seen.Hash); !errors.As(err, &ae) || ae.Code != apperr.ShortcutChanged {
		t.Fatalf("err = %v, want %s", err, apperr.ShortcutChanged)
	}
	if _, err := s.Inspect(ctx, id, dir+"gone.sh"); !errors.As(err, &ae) || ae.Code != apperr.NotFound {
		t.Fatalf("err = %v, want %s", err, apperr.NotFound)
	}
}

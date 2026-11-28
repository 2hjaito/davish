package fs

import (
	"bytes"
	"context"
	"sort"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Reading as root (issue #177): a folder or file the login user may not
// read can still be shown, after the user saw the exact command. Both
// commands only read; neither follows a path through anything Davish
// has not been given, and both are bounded in output.

// notFile is the exit status of a root read that finds no regular file;
// no program in the line exits with it.
const notFile = 74

// maxList bounds what a root listing sends back.
const maxList = 8 << 20

// listFormat is find's -printf: type, target type, size, mtime, mode,
// uid, gid and name (which may hold a tab) end one NUL-terminated
// field; the link target is the next one.
const listFormat = `%y\t%Y\t%s\t%T@\t%m\t%U\t%G\t%f\0%l\0`

func listCommand(dir string) string {
	return shellx.Cmd("find", "-P", dir, "-mindepth", "1", "-maxdepth", "1", "-printf", listFormat).String() +
		" | head -c " + strconv.Itoa(maxList)
}

// readCommand prints the file's size, mtime, mode, owner and group on
// one line, then up to MaxRead+1 bytes of it. The path may be a link:
// what it leads to is read, as Read does.
func readCommand(p string) string {
	q := shellx.Quote(p)
	return "test -f " + q + " || exit " + strconv.Itoa(notFile) + "; " +
		"stat -L -c '%s %Y %a %u %g' -- " + q + " && head -c " + strconv.Itoa(MaxRead+1) + " -- " + q
}

// RootReadPlan is what the confirmation shows before a read as root.
type RootReadPlan struct {
	// Sudo is the command as it runs through sudo.
	Sudo string `json:"sudo"`
}

// PlanListAsRoot gives the command ListAsRoot runs, for the confirmation.
func (s *Service) PlanListAsRoot(dir string) (RootReadPlan, error) {
	if err := CheckPath(dir); err != nil {
		return RootReadPlan{}, err
	}
	return RootReadPlan{Sudo: shellx.Cmd("sudo", "sh", "-c", listCommand(dir)).String()}, nil
}

// PlanReadAsRoot gives the command ReadAsRoot runs, for the confirmation.
func (s *Service) PlanReadAsRoot(p string) (RootReadPlan, error) {
	if err := CheckPath(p); err != nil {
		return RootReadPlan{}, err
	}
	return RootReadPlan{Sudo: shellx.Cmd("sudo", "sh", "-c", readCommand(p)).String()}, nil
}

// ListAsRoot lists dir as root, like List. Errors are the sudo keeper's:
// SUDO_PASSWORD_REQUIRED, SUDO_PASSWORD_WRONG, CMD_NEEDS_ROOT.
func (s *Service) ListAsRoot(ctx context.Context, serverID, dir string) ([]Entry, error) {
	if err := CheckPath(dir); err != nil {
		return nil, err
	}
	if s.root == nil {
		return nil, apperr.New(apperr.CmdNeedsRoot, "no way to root")
	}
	res, err := s.root.Run(ctx, serverID, "files", listCommand(dir), nil)
	if err != nil {
		return nil, err
	}
	// find fails as a whole only when dir itself cannot be listed; the
	// pipe hides that, so an empty answer is checked below.
	entries := parseList(dir, res.Stdout)
	if len(entries) == 0 && len(bytes.TrimSpace(res.Stderr)) > 0 {
		res.Exit = 1
		return nil, exitError(res)
	}
	return entries, nil
}

// parseList reads listCommand's output. A last record cut by the size
// limit is dropped.
func parseList(dir string, out []byte) []Entry {
	fields := bytes.Split(out, []byte{0})
	var list []Entry
	for i := 0; i+1 < len(fields); i += 2 {
		meta, target := string(fields[i]), string(fields[i+1])
		parts := strings.SplitN(meta, "\t", 8)
		if len(parts) != 8 || parts[7] == "" {
			continue
		}
		size, _ := strconv.ParseInt(parts[2], 10, 64)
		mtime, _, _ := strings.Cut(parts[3], ".")
		sec, _ := strconv.ParseInt(mtime, 10, 64)
		perm, _ := strconv.ParseUint(parts[4], 8, 32)
		uid, _ := strconv.ParseUint(parts[5], 10, 32)
		gid, _ := strconv.ParseUint(parts[6], 10, 32)
		kind := kindOfFind(parts[0])
		mode := findType(parts[0]) | uint32(perm) //nolint:gosec // a mode fits 32 bits
		e := Entry{
			Name: parts[7], Path: strings.TrimSuffix(dir, "/") + "/" + parts[7], Kind: kind,
			Size: size, Mode: lsMode(mode), Perm: uint32(perm) & 0o7777, ModTime: sec, //nolint:gosec // as above
			UID: uint32(uid), GID: uint32(gid), //nolint:gosec // ids fit 32 bits
		}
		if kind == KindLink {
			e.Target = target
			e.Dir = parts[1] == "d"
		}
		e.Dir = e.Dir || kind == KindDir
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

func findType(y string) uint32 {
	switch y {
	case "f":
		return sIFREG
	case "d":
		return sIFDIR
	case "l":
		return sIFLNK
	case "c":
		return sIFCHR
	case "b":
		return sIFBLK
	case "p":
		return sIFIFO
	case "s":
		return sIFSOCK
	}
	return 0
}

func kindOfFind(y string) string { return kindOf(findType(y)) }

// ReadAsRoot reads the file at p as root, like Read. The file's entry
// comes from stat, so a later save can tell whether it changed.
func (s *Service) ReadAsRoot(ctx context.Context, serverID, p string) (File, error) {
	if err := CheckPath(p); err != nil {
		return File{}, err
	}
	if s.root == nil {
		return File{}, apperr.New(apperr.CmdNeedsRoot, "no way to root")
	}
	res, err := s.root.Run(ctx, serverID, "files", readCommand(p), nil)
	if err != nil {
		return File{}, err
	}
	if res.Exit == notFile {
		return File{}, apperr.New(apperr.InvalidInput, "path is not a regular file")
	}
	if err := exitError(res); err != nil {
		return File{}, err
	}
	head, data, ok := bytes.Cut(res.Stdout, []byte("\n"))
	f := strings.Fields(string(head))
	if !ok || len(f) != 5 {
		return File{}, apperr.New(apperr.Internal, "unexpected answer from stat")
	}
	size, _ := strconv.ParseInt(f[0], 10, 64)
	mtime, _ := strconv.ParseInt(f[1], 10, 64)
	perm, _ := strconv.ParseUint(f[2], 8, 32)
	uid, _ := strconv.ParseUint(f[3], 10, 32)
	gid, _ := strconv.ParseUint(f[4], 10, 32)
	if size > MaxRead || len(data) > MaxRead {
		return File{}, apperr.New(apperr.FSTooLarge, "file is too large to open")
	}
	name := p[strings.LastIndexByte(p, '/')+1:]
	return File{
		Entry: Entry{
			Name: name, Path: p, Kind: KindFile, Size: size, Mode: lsMode(sIFREG | uint32(perm)), //nolint:gosec // a mode fits 32 bits
			Perm: uint32(perm) & 0o7777, ModTime: mtime, UID: uint32(uid), GID: uint32(gid), //nolint:gosec // ids fit 32 bits
		},
		Data: data,
	}, nil
}

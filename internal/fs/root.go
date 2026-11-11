package fs

import (
	"context"
	"errors"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// Root runs a command as root for a user who is not (sudo.Keeper).
type Root interface {
	Run(ctx context.Context, serverID, app, line string, stdin []byte) (audit.Result, error)
}

// UseRoot lets Files delete, and Editor save, what the server refuses the
// user, through r (ADR 0018): the same confirmed command again as root.
func (s *Service) UseRoot(r Root) { s.root = r }

// As root a mistake can break the server, so some paths are never removed
// or written that way, whatever the user confirms.
var (
	// rootTrees are refused with everything in them: the kernel's views,
	// the boot files, the system's programs and the package databases.
	rootTrees = []string{
		"/bin", "/boot", "/dev", "/lib", "/lib32", "/lib64", "/libx32", "/proc", "/run",
		"/sbin", "/sys", "/usr",
		"/var/lib/apt", "/var/lib/dpkg", "/var/lib/rpm",
	}
	// rootKeep are folders that stay, though what is in them may go.
	rootKeep = []string{
		"/", "/etc", "/home", "/media", "/mnt", "/opt", "/root", "/snap", "/srv", "/tmp",
		"/var", "/var/backups", "/var/cache", "/var/lib", "/var/log", "/var/mail", "/var/spool",
		"/var/tmp", "/var/www",
	}
	// rootVital are the files and folders without which nobody logs in.
	rootVital = []string{
		"/etc/fstab", "/etc/group", "/etc/gshadow", "/etc/passwd", "/etc/shadow",
		"/etc/ssh", "/etc/sudoers", "/etc/sudoers.d", "/etc/pam.d",
	}
)

func within(p string, trees []string) bool {
	return slices.ContainsFunc(trees, func(t string) bool { return p == t || strings.HasPrefix(p, t+"/") })
}

// RootRemoveRefused tells whether Files refuses to remove p as root.
func RootRemoveRefused(p string) bool {
	p = path.Clean(p)
	return within(p, rootTrees) || slices.Contains(rootKeep, p) || path.Dir(p) == "/home" ||
		within(p, rootVital)
}

// RootWriteRefused tells whether Editor refuses to save p as root. The
// vital files may be edited (that is what sudoedit is for), but not the
// kernel's views or the system's programs.
func RootWriteRefused(p string) bool {
	return within(path.Clean(p), rootTrees)
}

// trusted checks p before Davish acts on it as root. The folders on its
// way, and p itself when it is a link, must belong to root or to the
// login user: anyone else could swap one for a link between the
// confirmation and the command, and send root elsewhere. refused is then
// asked of p as written and of where it really is, so a link such as
// /var/run -> /run cannot slip past it.
func trusted(sc *sftp.Client, p string, refused func(string) bool) error {
	p = path.Clean(p)
	login, err := loginUID(sc)
	if err != nil {
		return err
	}
	owned := func(q string) error {
		fi, err := sc.Lstat(q)
		if err != nil {
			return mapErr("check", err)
		}
		st, ok := fileStat(fi)
		if !ok {
			return apperr.New(apperr.Internal, "server sent no file attributes")
		}
		if st.UID != 0 && st.UID != login {
			return apperr.New(apperr.FSProtected, "a folder on the way belongs to another user")
		}
		return nil
	}
	if fi, err := sc.Lstat(p); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if err := owned(p); err != nil {
			return err
		}
	}
	for d := path.Dir(p); ; d = path.Dir(d) {
		if err := owned(d); err != nil {
			return err
		}
		if d == "/" {
			break
		}
	}
	real, err := sc.RealPath(path.Dir(p))
	if err != nil {
		return mapErr("check", err)
	}
	if refused(p) || refused(path.Join(real, path.Base(p))) {
		return protected()
	}
	return nil
}

// loginUID is the login user's uid: the owner of their home folder.
func loginUID(sc *sftp.Client) (uint32, error) {
	home, err := sc.Getwd()
	if err != nil {
		return 0, apperr.Wrap(apperr.Internal, "cannot find the home folder", err)
	}
	fi, err := sc.Stat(home)
	if err != nil {
		return 0, mapErr("check", err)
	}
	st, ok := fileStat(fi)
	if !ok {
		return 0, apperr.New(apperr.Internal, "server sent no file attributes")
	}
	return st.UID, nil
}

func protected() error {
	return apperr.New(apperr.FSProtected, "Davish does not change this path as root")
}

// removeAsRoot runs the confirmed remove command as root once the server
// refused it to the user.
func (s *Service) removeAsRoot(ctx context.Context, serverID, p, command string) error {
	if s.root == nil {
		return apperr.New(apperr.FSPermissionDenied, "permission denied")
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return err
	}
	if err := trusted(sc, p, RootRemoveRefused); err != nil {
		return err
	}
	res, err := s.root.Run(ctx, serverID, "files", command, nil)
	if err != nil {
		if codeOf(err) == apperr.CmdNeedsRoot {
			return apperr.New(apperr.FSPermissionDenied, "permission denied")
		}
		return err
	}
	return exitError(res)
}

// RootSavePlan is what Editor shows before saving a file as root.
type RootSavePlan struct {
	// Command is exactly what runs as root; the new content goes in on
	// stdin.
	Command string `json:"command"`
	// Sudo is Command as it runs through sudo, for the confirmation.
	Sudo string `json:"sudo"`
	// Target is the file written: p, or what p links to.
	Target string `json:"target"`
	// Backup is the copy made first, if asked.
	Backup string `json:"backup,omitempty"`
}

// PlanSaveAsRoot works out how p would be saved as root, without saving.
// The file must exist: its owner and mode are kept, and the user must be
// able to read it, as Editor showed it.
func (s *Service) PlanSaveAsRoot(_ context.Context, serverID, p string, opts SaveOptions) (RootSavePlan, error) {
	return s.planSaveAsRoot(serverID, p, opts, time.Now())
}

func (s *Service) planSaveAsRoot(serverID, p string, opts SaveOptions, at time.Time) (RootSavePlan, error) {
	if err := CheckPath(p); err != nil {
		return RootSavePlan{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return RootSavePlan{}, err
	}
	target, orig, err := writeTarget(sc, p)
	if err != nil {
		return RootSavePlan{}, err
	}
	if orig == nil {
		return RootSavePlan{}, apperr.New(apperr.NotFound, "no such file")
	}
	// Both the path as opened and the file it leads to.
	if err := trusted(sc, p, RootWriteRefused); err != nil {
		return RootSavePlan{}, err
	}
	if err := trusted(sc, target, RootWriteRefused); err != nil {
		return RootSavePlan{}, err
	}
	if e := opts.Expect; e != nil && (int64(orig.Mtime) != e.ModTime || int64(orig.Size) != e.Size) { //nolint:gosec // sizes and times fit int64
		return RootSavePlan{}, apperr.New(apperr.FSChanged, "the file changed on the server since it was opened")
	}
	plan := RootSavePlan{Target: target}
	if opts.Backup {
		plan.Backup = backupName(target, at)
	}
	plan.Command = rootSaveCommand(target, plan.Backup, orig.UID, orig.GID, orig.Mode)
	plan.Sudo = shellx.Cmd("sudo", "sh", "-c", plan.Command).String()
	return plan, nil
}

// backupExists is the exit status of a root save that finds its backup
// name taken; no program in the line exits with it.
const backupExists = 73

// rootSaveCommand writes stdin over target as Write does, as root: into
// a temporary sibling mktemp creates (never through a link planted
// there), given target's owner and mode, then renamed over it. backup,
// when set, is a copy kept first, never over an existing file. Any
// failure leaves target as it was and removes the temporary file.
func rootSaveCommand(target, backup string, uid, gid, mode uint32) string {
	q := shellx.Quote
	dir, base := path.Dir(target), path.Base(target)
	var b strings.Builder
	b.WriteString("set -e; ")
	if backup != "" {
		b.WriteString("if test -e " + q(backup) + "; then exit " + strconv.Itoa(backupExists) + "; fi; ")
		b.WriteString("cp -p -- " + q(target) + " " + q(backup) + "; ")
	}
	b.WriteString(`t=$(mktemp ` + q(dir+"/."+base+".davish-XXXXXX") + `); trap 'rm -f -- "$t"' EXIT; `)
	b.WriteString(`cat > "$t"; `)
	b.WriteString(`chown ` + strconv.FormatUint(uint64(uid), 10) + ":" + strconv.FormatUint(uint64(gid), 10) + ` -- "$t"; `)
	b.WriteString(`chmod ` + strconv.FormatUint(uint64(mode&0o7777), 8) + ` -- "$t"; `)
	b.WriteString(`mv -f -- "$t" ` + q(target))
	return b.String()
}

// SaveAsRoot saves data over p as root exactly as confirmed: the plan is
// worked out again with the confirmed backup name and must give the same
// command, so a file that changed owner, mode or target meanwhile is
// refused with FS_STALE.
func (s *Service) SaveAsRoot(ctx context.Context, serverID, p string, data []byte, opts SaveOptions, confirmed RootSavePlan) (SaveResult, error) {
	if s.root == nil {
		return SaveResult{}, apperr.New(apperr.CmdNeedsRoot, "no way to root")
	}
	at := time.Now()
	if opts.Backup {
		stamp, ok := strings.CutPrefix(confirmed.Backup, confirmed.Target+".bak-")
		t, err := time.Parse("20060102T150405Z", stamp)
		if !ok || err != nil {
			return SaveResult{}, apperr.New(apperr.InvalidInput, "not a backup name")
		}
		at = t
	}
	plan, err := s.planSaveAsRoot(serverID, p, opts, at)
	if err != nil {
		return SaveResult{}, err
	}
	if plan.Command != confirmed.Command {
		return SaveResult{}, apperr.New(apperr.FSStale, "the file changed since the save was confirmed")
	}
	s.log.Info("save as root", "server", serverID, "app", "editor")
	res, err := s.root.Run(ctx, serverID, "editor", plan.Command, data)
	if err != nil {
		return SaveResult{}, err
	}
	if plan.Backup != "" && res.Exit == backupExists {
		return SaveResult{}, apperr.New(apperr.FSExists, "a backup of that name exists")
	}
	if err := exitError(res); err != nil {
		return SaveResult{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return SaveResult{}, err
	}
	e, err := stat(sc, plan.Target)
	if err != nil {
		return SaveResult{}, err
	}
	return SaveResult{Entry: e, Backup: plan.Backup}, nil
}

func codeOf(err error) apperr.Code {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

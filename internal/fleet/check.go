package fleet

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"

	"davish/internal/apperr"
	"davish/internal/packages"
)

// TypedFrom is the number of servers from which the user must type the
// count to confirm a run.
const TypedFrom = 3

// Check is what the confirm dialog needs to know about a command.
type Check struct {
	// Destructive is set when the command looks like it removes, stops or
	// overwrites something. Only a warning: every run is confirmed.
	Destructive bool `json:"destructive"`
	// Typed asks the user to type the number of servers to confirm.
	Typed bool `json:"typed"`
}

// CheckCommand tells how a run of command on n servers must be confirmed.
func CheckCommand(command string, n int) Check {
	d := Destructive(command)
	return Check{Destructive: d, Typed: d || n >= TypedFrom}
}

// destructive matches commands that remove, stop, kill, reboot, format
// or overwrite. A word must start a command: at the start, or after
// ; & | ( or sudo and its options.
var destructive = regexp.MustCompile(
	`(?:^|[;&|(]\s*|\bsudo\s+(?:-\S+\s+)*)` +
		`(?:rm|rmdir|shred|wipefs|reboot|shutdown|poweroff|halt|init\s+[06]|` +
		`mkfs(?:\.\w+)?|dd|fdisk|parted|truncate|kill|pkill|killall|` +
		`systemctl\s+(?:\S+\s+)*?(?:stop|restart|disable|mask|isolate|reboot|poweroff|halt)|` +
		`service\s+\S+\s+(?:stop|restart)|` +
		`docker\s+(?:rm|rmi|kill|stop|system\s+prune|(?:container|image|volume|network)\s+(?:rm|prune|kill|stop))|` +
		`userdel|deluser|chmod\s+-R|chown\s+-R|crontab\s+-r)` +
		`(?:\s|$|;|&|\|)`)

// overwrite matches "> file" that is not "2>", ">>", ">&" or /dev/null.
var overwrite = regexp.MustCompile(`(?:^|[^>0-9&])>\s*([^\s&>|;]+)`)

// Destructive reports whether command looks like it destroys something.
func Destructive(command string) bool {
	c := strings.TrimSpace(command)
	if destructive.MatchString(c) {
		return true
	}
	for _, m := range overwrite.FindAllStringSubmatch(c, -1) {
		if m[1] != "/dev/null" {
			return true
		}
	}
	return false
}

// Detector tells a server's package manager; packages.Service in the app.
type Detector interface {
	Detect(ctx context.Context, serverID string) (string, error)
}

// Skip is a server left out of a prepared run, with an apperr code or
// "noManager".
type Skip struct {
	ServerID string `json:"serverID"`
	Reason   string `json:"reason"`
}

// Update is the "Update packages" run: each server's own command.
type Update struct {
	Items   []Item `json:"items"`
	Skipped []Skip `json:"skipped"`
}

// PlanUpdate builds the package update command for each server, with
// sudo -n: Davish never types a sudo password for a run (docs/adr/0021),
// so a server whose sudo wants one fails and says why.
func PlanUpdate(ctx context.Context, d Detector, serverIDs []string) Update {
	type res struct {
		item Item
		skip *Skip
	}
	out := make([]res, len(serverIDs))
	slots := make(chan struct{}, Parallel)
	var wg sync.WaitGroup
	for i, id := range serverIDs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			m, err := d.Detect(ctx, id)
			if err != nil {
				out[i].skip = &Skip{ServerID: id, Reason: code(err)}
				return
			}
			line, err := packages.PlanAction(m, packages.OpUpgradeAll, nil)
			if err != nil {
				out[i].skip = &Skip{ServerID: id, Reason: "noManager"}
				return
			}
			out[i].item = Item{ServerID: id, Command: nonInteractiveSudo(line)}
		}(i, id)
	}
	wg.Wait()
	u := Update{Items: []Item{}, Skipped: []Skip{}}
	for _, r := range out {
		if r.skip != nil {
			u.Skipped = append(u.Skipped, *r.skip)
		} else {
			u.Items = append(u.Items, r.item)
		}
	}
	return u
}

// nonInteractiveSudo turns each "sudo " that PlanAction wrote into
// "sudo -n ". Its package names cannot hold a space, so every match is a
// command it built.
func nonInteractiveSudo(line string) string {
	if strings.HasPrefix(line, "sudo ") {
		line = "sudo -n " + line[len("sudo "):]
	}
	return strings.ReplaceAll(line, " && sudo ", " && sudo -n ")
}

func code(err error) string {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return string(ae.Code)
	}
	return string(apperr.Internal)
}

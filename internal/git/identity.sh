# Who the session is on the server and what git and the code hosts know it
# as, for identity.go. Read only: the pinned host keys go to a temporary
# file that is removed at the end, ~/.ssh is never written. $1... are
# "host key" lines of the hosts to ask.
export LC_ALL=C
echo '@@user'
id -un
echo '@@home'
printf '%s\n' "$HOME"
echo '@@host'
hostname 2>/dev/null
echo '@@gitname'
git config --global --get user.name 2>/dev/null
echo '@@gitemail'
git config --global --get user.email 2>/dev/null
echo '@@helper'
git config --global --get credential.helper 2>/dev/null
echo '@@keys'
for f in "$HOME"/.ssh/*.pub; do
	[ -f "$f" ] && ssh-keygen -lf "$f" 2>/dev/null
done
kh=$(mktemp 2>/dev/null) || exit 0
trap 'rm -f "$kh"' EXIT
for line in "$@"; do
	printf '%s\n' "$line" > "$kh"
	host=${line%% *}
	echo "@@host:$host"
	# BatchMode: a key the agent or ~/.ssh offers, never a prompt. Only the
	# published key is trusted, so a spoofed host fails instead of passing.
	timeout 10 ssh -T -o BatchMode=yes -o ConnectTimeout=6 -o StrictHostKeyChecking=yes \
		-o "UserKnownHostsFile=$kh" -o GlobalKnownHostsFile=/dev/null "git@$host" < /dev/null 2>&1
done

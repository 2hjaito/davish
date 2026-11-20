# Who the session is on the server and what git and the code hosts know it
# as, for identity.go. Read only: ~/.ssh is never written.
#
#   sh -s -- local          the user, git's name and email, the keys: fast
#   sh -s -- hosts LINE...  ssh -T to each host of the "host key" LINEs, all
#                           at once; the pinned keys go to a temporary folder
#                           that is removed at the end
export LC_ALL=C
if [ "$1" = local ]; then
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
	exit 0
fi
[ "$1" = hosts ] || exit 2
shift
d=$(mktemp -d 2>/dev/null) || exit 0
trap 'rm -rf "$d"' EXIT
for line in "$@"; do
	host=${line%% *}
	printf '%s\n' "$line" > "$d/$host.kh"
	# BatchMode: a key the agent or ~/.ssh offers, never a prompt. Only the
	# published key is trusted, so a spoofed host fails instead of passing.
	timeout 8 ssh -T -o BatchMode=yes -o ConnectTimeout=4 -o StrictHostKeyChecking=yes \
		-o "UserKnownHostsFile=$d/$host.kh" -o GlobalKnownHostsFile=/dev/null "git@$host" \
		< /dev/null > "$d/$host.out" 2>&1 &
done
wait
for line in "$@"; do
	host=${line%% *}
	echo "@@host:$host"
	cat "$d/$host.out" 2>/dev/null
done

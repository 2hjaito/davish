# Who the session is on the server and what git and the code hosts know it
# as, for identity.go. Read only: ~/.ssh is never written.
#
#   sh -s -- local       the user, git's name and email, the keys: fast
#   sh -s -- host LINE   ssh -T to the host of the "host key" LINE, trusting
#                        only that key, kept in a temporary file removed at
#                        the end
#
# Everything is in main, so the shell has read the whole script from stdin
# before anything runs and no command can read the rest of it.
main() {
	export LC_ALL=C
	case $1 in
	local)
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
		;;
	host)
		host=${2%% *}
		kh=$(mktemp 2>/dev/null) || return 0
		trap 'rm -f "$kh"' EXIT
		printf '%s\n' "$2" > "$kh"
		# Some systems have no timeout; ConnectTimeout still bounds a host
		# that never answers.
		t=
		command -v timeout > /dev/null 2>&1 && t='timeout 8'
		echo "@@host:$host"
		# BatchMode: a key the agent or ~/.ssh offers, never a prompt. Only
		# the published key is trusted, so a spoofed host fails.
		$t ssh -T -o BatchMode=yes -o ConnectTimeout=4 -o StrictHostKeyChecking=yes \
			-o "UserKnownHostsFile=$kh" -o GlobalKnownHostsFile=/dev/null "git@$host" < /dev/null 2>&1
		;;
	*) return 2 ;;
	esac
}
main "$@"

# Davish prompt, handed to `bash --rcfile` by term.Start (docs/adr/0004).
# Nothing is written to the server. Keep it working on bash 4.2 (CentOS 7).

# Load the user's environment as a login shell would; those files usually
# source ~/.bashrc themselves.
if [ -r /etc/profile ]; then . /etc/profile; fi
if [ -r ~/.bash_profile ]; then
	. ~/.bash_profile
elif [ -r ~/.bash_login ]; then
	. ~/.bash_login
elif [ -r ~/.profile ]; then
	. ~/.profile
elif [ -r ~/.bashrc ]; then
	. ~/.bashrc
fi

shopt -s checkwinsize

__davish_git=
command -v git >/dev/null 2>&1 && __davish_git=1

# Glyphs need a UTF-8 locale, or readline miscounts the prompt width.
# Davish's terminal is always UTF-8, but unlike the ssh command it does not
# send LANG, so pick an installed UTF-8 locale when none is set.
__davish_utf8=
case "${LC_ALL:-${LC_CTYPE:-${LANG:-}}}" in *[Uu][Tt][Ff]-8* | *[Uu][Tt][Ff]8*) __davish_utf8=1 ;; esac
if [ -z "$__davish_utf8" ] && [ -z "${LC_ALL:-}" ]; then
	for __davish_l in C.UTF-8 C.utf8 en_US.UTF-8 en_US.utf8; do
		if locale -a 2>/dev/null | grep -qx "$__davish_l"; then
			export LC_CTYPE=$__davish_l
			__davish_utf8=1
			break
		fi
	done
	unset __davish_l
fi

# Escape sequences are built with a real ESC so that names in the prompt
# (directories, branches) are printed as they are, never interpreted.
__davish_e=$'\e'

# Palette from DESIGN.md as truecolor "r;g;b".
__davish_dark='29;31;37'
__davish_user='126;226;168'
__davish_root='255;138;101'
__davish_path='138;180;255'
__davish_branch='255;184;107'
__davish_ok='126;226;168'
__davish_err='255;107;107'
__davish_muted='164;169;182'

# Start of the running command, in microseconds. The DEBUG trap fires before
# each command line and before the prompt function; only the first counts.
__davish_t0=
__davish_mark() {
	[ -n "$__davish_t0" ] && return
	if [ -n "${EPOCHREALTIME:-}" ]; then
		__davish_t0=${EPOCHREALTIME/[.,]/}
	else
		__davish_t0=$((SECONDS * 1000000))
	fi
}
trap '__davish_mark' DEBUG

__davish_duration() {
	local us=$1 s
	s=$((us / 1000000))
	if [ "$s" -ge 3600 ]; then
		printf '%dh %dm' $((s / 3600)) $((s % 3600 / 60))
	elif [ "$s" -ge 60 ]; then
		printf '%dm %ds' $((s / 60)) $((s % 60))
	else
		printf '%d.%ds' "$s" $((us % 1000000 / 100000))
	fi
}

# One coloured block; the arrow into it takes the previous block's colour.
__davish_seg() {
	local bg=$1 text=$2
	if [ -z "$__davish_prev" ]; then
		__davish_line+="${__davish_e}[38;2;${bg}m"$'\ue0b6'
	else
		__davish_line+="${__davish_e}[38;2;${__davish_prev};48;2;${bg}m"$'\ue0b0'
	fi
	__davish_line+="${__davish_e}[48;2;${bg};38;2;${__davish_dark}m ${text} "
	__davish_prev=$bg
	__davish_width=$((__davish_width + ${#text} + 3))
}

__davish_prompt() {
	local code=$? now elapsed=0 dir branch status=''
	if [ -n "${EPOCHREALTIME:-}" ]; then now=${EPOCHREALTIME/[.,]/}; else now=$((SECONDS * 1000000)); fi
	[ -n "$__davish_t0" ] && elapsed=$((now - __davish_t0))
	__davish_t0=

	if [ -z "$__davish_utf8" ]; then
		PS1='\[\e]0;\u@\h: \w\a\]\u@\h:\w\$ '
		return
	fi

	__davish_line='' __davish_prev='' __davish_width=0
	if [ "$EUID" -eq 0 ]; then
		__davish_seg "$__davish_root" "${USER:-root}@${HOSTNAME%%.*}"
	else
		__davish_seg "$__davish_user" "${USER:-$(id -un)}@${HOSTNAME%%.*}"
	fi
	dir=${PWD/#$HOME/\~}
	if [ "${#dir}" -gt 40 ]; then dir="…/${dir#"${dir%/*/*}"/}"; fi
	__davish_seg "$__davish_path" $'\uf07b'" $dir"
	if [ -n "$__davish_git" ]; then
		branch=$(git symbolic-ref --short -q HEAD 2>/dev/null || git rev-parse --short HEAD 2>/dev/null)
		[ -n "$branch" ] && __davish_seg "$__davish_branch" $'\ue0a0'" $branch"
	fi
	__davish_line+="${__davish_e}[0m${__davish_e}[38;2;${__davish_prev}m"$'\ue0b0'"${__davish_e}[0m"

	if [ "$code" -ne 0 ]; then
		status+="${__davish_e}[38;2;${__davish_err}m"$'\uf00d'" $code"
	fi
	if [ "$elapsed" -ge 1000000 ]; then
		[ "$code" -eq 0 ] && status+="${__davish_e}[38;2;${__davish_ok}m"$'\uf00c'
		status+="${__davish_e}[38;2;${__davish_muted}m $(__davish_duration "$elapsed")"
	fi
	[ -n "$status" ] && __davish_line+=" $status${__davish_e}[0m"

	# The clock sits on the right while it fits; status width is a rough guess.
	local clock
	printf -v clock '%(%H:%M)T' -1
	if [ $((__davish_width + 24)) -lt "${COLUMNS:-80}" ]; then
		__davish_line+="${__davish_e}[$((COLUMNS - 7))G${__davish_e}[38;2;${__davish_muted}m"$'\uf017'" ${clock}${__davish_e}[0m"
	fi
	printf '%s\n' "$__davish_line"

	if [ "$code" -eq 0 ]; then
		PS1="\[\e]0;\u@\h: \w\a\e[38;2;${__davish_ok}m\]"$'\u276f'"\[\e[0m\] "
	else
		PS1="\[\e]0;\u@\h: \w\a\e[38;2;${__davish_err}m\]"$'\u276f'"\[\e[0m\] "
	fi
}

PROMPT_COMMAND="__davish_prompt${PROMPT_COMMAND:+; $PROMPT_COMMAND}"

# Looks at a repository and what runs it, for deployinfo.go. $1 is the
# repository folder. Read only; every source is optional.
export LC_ALL=C
cd -- "$1" 2>/dev/null || exit 3
echo '@@uid'
id -u
echo '@@root'
for f in package.json pnpm-lock.yaml yarn.lock package-lock.json npm-shrinkwrap.json \
	docker-compose.yml docker-compose.yaml compose.yml compose.yaml Dockerfile go.mod Makefile; do
	[ -f "$f" ] && echo "$f"
done
echo '@@package'
[ -f package.json ] && head -c 262144 package.json
echo
echo '@@pm2'
p=$(command -v pm2 2>/dev/null)
if [ -z "$p" ]; then
	# Installed with nvm, which only login shells put on the PATH.
	for f in "$HOME"/.nvm/versions/node/*/bin/pm2 /usr/local/bin/pm2; do
		[ -x "$f" ] && p=$f && break
	done
fi
# pm2 is a node script: its own folder holds the node it needs.
[ -n "$p" ] && PATH="$(dirname "$p"):$PATH" "$p" jlist 2>/dev/null
echo
echo '@@systemd'
units=$(systemctl list-units --type=service --state=running --no-legend --plain 2>/dev/null | awk '{print $1}')
# shellcheck disable=SC2086 # one word per unit
[ -n "$units" ] && systemctl show -p Id -p WorkingDirectory -p ExecStart -- $units 2>/dev/null
echo '@@compose'
docker ps --filter "label=com.docker.compose.project.working_dir=$1" \
	--format '{{.Label "com.docker.compose.project"}}' 2>/dev/null | sort -u

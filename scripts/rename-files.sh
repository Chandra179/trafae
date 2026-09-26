#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "Usage: $0 <new-module-name>"
	echo "Example: $0 github.com/me/myproject"
	echo ""
	echo "Renames the current Go module and the project's Lux references to the"
	echo "last path segment of <new-module-name>."
	exit 1
}

if [ $# -ne 1 ]; then
	usage
fi

if [ ! -f go.mod ]; then
	echo "go.mod not found; run this script from the repository root." >&2
	exit 1
fi

old="$(sed -n 's/^module //p' go.mod)"
new="$1"
old_name="${old##*/}"
new_name="${new##*/}"
old_title="$(sed -n 's|^// @title[[:space:]]*\(.*\) API$|\1|p' server/cmd/example/main.go | head -n 1)"
new_title="$(tr '[:lower:]' '[:upper:]' <<<"${new_name:0:1}")${new_name:1}"

if [ "$old" = "$new" ]; then
	echo "Old and new module names are identical. Nothing to do."
	exit 0
fi

echo "Renaming module '$old' -> '$new' (project name '$old_name' -> '$new_name')"

# 1. Update go.mod module line.
sed -i "s|^module $old\$|module $new|" go.mod
echo "  updated go.mod"

# 2. Update import paths in repository Go files, excluding generated/vendor
#    dependencies and Git metadata.
find . -path './vendor' -prune -o -path './.git' -prune -o -type f -name '*.go' -exec sed -i "s|\"$old/|\"$new/|g" {} +
echo "  updated import paths in .go files"

# 3. Also update proto files if any exist.
proto_matches="$(find . -type f -name '*.proto' -exec grep -l "$old/" {} + 2>/dev/null || true)"
if [ -n "$proto_matches" ]; then
	find . -name '*.proto' -exec sed -i "s|$old/|$new/|g" {} +
	echo "  updated .proto files"
fi

# 4. Update local-prefixes in .golangci.yml.
sed -i "s|^\(\s*- \)$old$|\1$new|" .golangci.yml
echo "  updated .golangci.yml"

# 5. Update mockery package keys, which mirror the full module path.
sed -i \
	-e "s|^\(\s*\)$old:$|\1$new:|" \
	-e "s|^\(\s*\)$old/|\1$new/|" \
	.mockery.yaml
echo "  updated .mockery.yaml"

# 6. Update the Swagger title in the command entrypoint.
sed -i "s|@title           $old_title API|@title           $new_title API|" server/cmd/example/main.go
echo "  updated Swagger title in server/cmd/example/main.go"

# 7. Rename the command directory when the caller is renaming the project.
if [ -d "server/cmd/example" ]; then
	mv server/cmd/example "server/cmd/$new_name"
	echo "  renamed server/cmd/example -> server/cmd/$new_name"
fi

# 8. Update embedded-store project naming in environment/config/CI files.
for f in .env.example .github/workflows/ci.yml; do
	[ -f "$f" ] || continue
	sed -i \
		-e "s|^SQLITE_DSN=$old_name\.db|SQLITE_DSN=$new_name.db|" \
		-e "s|^BADGER_DIR=$old_name|BADGER_DIR=$new_name|" \
		"$f"
done
for f in server/config/config_dev.yaml server/config/config_prd.yaml; do
	[ -f "$f" ] || continue
	sed -i \
		-e "s|dsn: \"$old_name\.db\"|dsn: \"$new_name.db\"|" \
		-e "s|dir: \"$old_name\"|dir: \"$new_name\"|" \
		"$f"
done
echo "  updated environment and datastore names"

# 9. Update project documentation and Containerfile/Makefile entrypoint paths.
find . -path './vendor' -prune -o -path './.git' -prune -o -type f -name '*.md' -exec sed -i \
	-e "s|$old/|$new/|g" \
	-e "s|server/cmd/example|server/cmd/$new_name|g" \
	-e "s|\<Lux\>|$new_title|g" \
	-e "s|\<lux\>|$new_name|g" \
	{} +
sed -i "s|server/cmd/example|server/cmd/$new_name|g" Containerfile Makefile
echo "  updated documentation, Containerfile, and Makefile"

echo "Done. Run 'go test ./...' and 'go build ./...' to verify the renamed module."

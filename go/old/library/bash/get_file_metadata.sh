#!/bin/bash
## Function to inspect and display various metadata of a given file
inspect_file_meta() {
	local f="$1"
	if [[ -z "$f" ]]; then
		echo "Usage: inspect_file_meta /path/to/file" >&2
		return 1
	fi
	if [[ ! -e "$f" ]]; then
		echo "No such file: $f" >&2
		return 1
	fi

	echo "=== stat ==="
	stat -c 'File: %n
Size: %s
UID:  %u (%U)
GID:  %g (%G)
Mode: %a (%A)
Access (atime): %x
Modify (mtime): %y
Change (ctime): %z' -- "$f"

	echo
	echo "=== SELinux context (if enabled) ==="
	# -Z prints context; -d avoids recursing if it's a dir
	ls -Zd -- "$f" 2> /dev/null || echo "(no SELinux or ls -Z not available)"

	echo
	echo "=== ACLs (if any) ==="
	getfacl -p -- "$f" 2> /dev/null || echo "(no ACLs or getfacl not installed)"

	echo
	echo "=== Extended attributes (xattrs) ==="
	# -d: dump attrs, -m-: match all, --absolute-names: show full path
	getfattr -d -m- --absolute-names -- "$f" 2> /dev/null || echo "(no xattrs or getfattr not installed)"

	echo
	echo "=== File flags (immutable/append-only etc.) ==="
	lsattr -d -- "$f" 2> /dev/null || echo "(lsattr not available or filesystem doesn’t support flags)"
}

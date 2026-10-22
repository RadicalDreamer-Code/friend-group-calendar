#!/bin/sh
# Dumps the database once a day at BACKUP_HOUR_UTC into /backups and deletes
# dumps older than BACKUP_KEEP_DAYS. Runs in the postgres image (pg_dump and
# busybox); connection settings come from the PG* variables.
#
# One backup right away:  docker compose exec backup sh /backup.sh now
set -eu

hour=${BACKUP_HOUR_UTC:-3}
keep=${BACKUP_KEEP_DAYS:-14}

backup() {
	name="calendar-$(date -u +%Y-%m-%d-%H%M).dump"
	# Written under a temporary name, so a dump that fails halfway never looks complete.
	if pg_dump --format=custom --file="/backups/$name.partial"; then
		mv "/backups/$name.partial" "/backups/$name"
		echo "backup: wrote $name ($(du -h "/backups/$name" | cut -f1))"
	else
		rm -f "/backups/$name.partial"
		echo "backup: pg_dump failed" >&2
	fi
	find /backups -name 'calendar-*.dump' -mtime +"$keep" -print -delete | sed 's/^/backup: deleted /'
}

seconds_until_next_run() {
	now=$(date -u +%s)
	next=$(date -u -d "$(date -u +%Y-%m-%d) $(printf '%02d' "$hour"):00:00" +%s)
	[ "$next" -gt "$now" ] || next=$((next + 86400))
	echo $((next - now))
}

if [ "${1:-}" = now ]; then
	backup
	exit
fi

echo "backup: daily at $(printf '%02d' "$hour"):00 UTC, keeping $keep days"
while true; do
	sleep "$(seconds_until_next_run)"
	backup
done

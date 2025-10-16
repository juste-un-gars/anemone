#!/bin/bash
echo "Setup CRON"
echo "0 2 * * * /scripts/backup-now.sh >> /logs/backup.log 2>&1" > /etc/crontabs/root

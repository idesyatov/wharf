#!/bin/sh
# Wharf — stress test script for chart visualization
# Usage: docker cp scripts/stress.sh <container>:/tmp/
#        docker exec <container> sh /tmp/stress.sh [duration_sec] [mem_step_mb]

DURATION="${1:-60}"
MEM_STEP_MB="${2:-2}"

# Cleanup on exit
cleanup() {
    kill 0 2>/dev/null
}
trap cleanup EXIT INT TERM

echo "=== Stress: ${DURATION}s, MEM step ${MEM_STEP_MB}MB ==="

START=$(date +%s)

# CPU: wave pattern — 3s work, 2s pause
cpu_wave() {
    while true; do
        END_WORK=$(($(date +%s) + 3))
        while [ "$(date +%s)" -lt "$END_WORK" ]; do
            : $((1 + 1))
        done
        sleep 2
    done
}

cpu_wave &

# MEM: allocate via separate shell variables (each ~MEM_STEP_MB)
# /dev/zero + tr is the most portable approach (works in any busybox)
BLOCK_KB=$((MEM_STEP_MB * 1024))
i=0
while true; do
    NOW=$(date +%s)
    ELAPSED=$((NOW - START))
    if [ "$ELAPSED" -ge "$DURATION" ]; then
        break
    fi

    eval "MEMBLK_$i=\"$(dd if=/dev/zero bs=1024 count="$BLOCK_KB" 2>/dev/null | tr '\0' 'A')\""
    i=$((i + 1))
    echo "allocated block $i (~${MEM_STEP_MB}MB, total ~$((i * MEM_STEP_MB))MB)"
    sleep 2
done

echo "=== Done ==="

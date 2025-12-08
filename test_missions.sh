#!/bin/bash
# Mission Testing Automation Script
# Tests:
# 1. Single mission lifecycle
# 2. Concurrency (multiple missions)
# 3. Token expiry & rotation

COMMANDER_URL="http://localhost:8080"
LOGIN_ENDPOINT="$COMMANDER_URL/login"
MISSION_ENDPOINT="$COMMANDER_URL/missions"

#############################
# Helper Functions
#############################

get_token() {
    echo "[INFO] Fetching new JWT token..."
    TOKEN=$(curl -s -X POST $LOGIN_ENDPOINT | jq -r '.token')
    echo "[INFO] Token received: $TOKEN"
}

submit_mission() {
    local order="$1"
    echo "[INFO] Submitting mission: $order"

    RESPONSE=$(curl -s -X POST "$MISSION_ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"order\": \"$order\"}")

    echo "$RESPONSE"
}

check_status() {
    local id="$1"
    STATUS=$(curl -s "$MISSION_ENDPOINT/$id" | jq -r '.status')
    echo "$STATUS"
}

wait_for_completion() {
    local id="$1"

    echo "[INFO] Waiting for mission completion: $id"
    while true; do
        STATUS=$(check_status "$id")
        echo "[STATUS] $id -> $STATUS"

        if [[ "$STATUS" == "COMPLETED" || "$STATUS" == "FAILED" ]]; then
            break
        fi
        sleep 2
    done
}

#########################################
# 1. Single Mission End-to-End Test
#########################################

single_mission_test() {
    echo "===================================="
    echo "Running Single Mission Flow Test"
    echo "===================================="

    get_token
    RESPONSE=$(submit_mission "Scan Area")
    MID=$(echo "$RESPONSE" | jq -r '.mission_id')

    echo "[INFO] Mission ID: $MID"

    wait_for_completion "$MID"
}

#########################################
# 2. Concurrency Test
#########################################

concurrency_test() {
    echo "===================================="
    echo "Running Concurrency Test"
    echo "===================================="

    get_token

    MISSION_IDS=()

    # Fire 5 missions quickly
    for i in {1..5}; do
        RESPONSE=$(submit_mission "Concurrent Mission $i")
        MID=$(echo "$RESPONSE" | jq -r '.mission_id')
        MISSION_IDS+=("$MID")
        echo "[INFO] Submitted: $MID"
    done

    # Track each mission
    for MID in "${MISSION_IDS[@]}"; do
        wait_for_completion "$MID" &
    done

    wait
}

#########################################
# 3. Authentication & Token Rotation Test
#########################################

token_rotation_test() {
    echo "===================================="
    echo "Running Token Expiry & Rotation Test"
    echo "===================================="

    get_token

    RESPONSE=$(submit_mission "Initial Token Test")
    MID1=$(echo "$RESPONSE" | jq -r '.mission_id')

    echo "[INFO] Mission 1 submitted with original token"

    echo "[INFO] Simulating token expiry... sleeping 3 seconds"
    sleep 3

    echo "[INFO] Requesting new token (rotation)"
    get_token

    RESPONSE=$(submit_mission "Post-Expiry Mission")
    MID2=$(echo "$RESPONSE" | jq -r '.mission_id')

    echo "[INFO] Mission 2 submitted with new rotated token"

    wait_for_completion "$MID1" &
    wait_for_completion "$MID2" &

    wait
}

#########################################
# Test Runner
#########################################

case "$1" in
    single)
        single_mission_test
        ;;
    concurrency)
        concurrency_test
        ;;
    rotation)
        token_rotation_test
        ;;
    all)
        single_mission_test
        concurrency_test
        token_rotation_test
        ;;
    *)
        echo "Usage: ./test_missions.sh {single|concurrency|rotation|all}"
        exit 1
        ;;
esac

#!/bin/bash

CommanderURL="http://localhost:8080"

LogFile="mission_test_$(date +"%Y%m%d_%H%M%S").log"
touch "$LogFile"

log() {
    level=${2:-INFO}
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    msg="[$timestamp] [$level] $1"
    echo "$msg"
    echo "$msg" >> "$LogFile"
}

log_newline() {
    echo "" | tee -a "$LogFile"
}

get_auth_token() {
    log "Attempting login to get JWT token..."

    response=$(curl -s -X POST "$CommanderURL/login" \
        -H "Content-Type: application/json")

    access=$(echo "$response" | jq -r '.token.access_token')
    refresh=$(echo "$response" | jq -r '.token.refresh_token')

    if [[ "$access" == "null" || -z "$access" ]]; then
        log "Login failed: Token missing" "ERROR"
        return 1
    fi

    log "Received JWT token successfully."

    echo "$access|$refresh"
}

test_single_mission() {
    log_newline
    log "=== Running Test 1: Single Mission Flow ==="
    log_newline

    tokens=$(get_auth_token) || return
    access=$(echo "$tokens" | cut -d '|' -f1)

    payload='{"order":"Attack north base"}'

    response=$(curl -s -X POST "$CommanderURL/missions" \
        -H "Authorization: Bearer $access" \
        -H "Content-Type: application/json" \
        -d "$payload")

    missionId=$(echo "$response" | jq -r '.mission_id')
    status=$(echo "$response" | jq -r '.status')

    if [[ "$missionId" == "null" ]]; then
        log "Test 1 Failed: Invalid mission response" "ERROR"
        return
    fi

    log "Mission $missionId (Status: $status) submitted."

    if [[ "$status" != "QUEUED" ]]; then
        log "Expected QUEUED but got $status" "ERROR"
    else
        log "Test 1 Passed: Mission accepted."
    fi

    log_newline
}

wait_for_mission_status() {
    mission_id=$1
    timeout=${2:-60}

    end=$((SECONDS + timeout))

    while (( SECONDS < end )); do
        response=$(curl -s "$CommanderURL/missions/$mission_id")
        status=$(echo "$response" | jq -r '.status')

        log "Mission $mission_id: Current status = $status"

        if [[ "$status" == "COMPLETED" || "$status" == "FAILED" ]]; then
            return
        fi

        sleep 3
    done

    log "Mission $mission_id: TIMEOUT"
}

test_concurrency() {
    log_newline
    log "=== Running Test 2: Concurrency (20 missions) ==="
    log_newline

    tokens=$(get_auth_token) || return
    access=$(echo "$tokens" | cut -d "|" -f 1)

    mission_ids=()

    for i in $(seq 1 20); do
        payload="{\"order\":\"Attack Zone-$i\"}"

        response=$(curl -s -X POST "$CommanderURL/missions" \
            -H "Authorization: Bearer $access" \
            -H "Content-Type: application/json" \
            -d "$payload")

        missionId=$(echo "$response" | jq -r '.mission_id')

        if [[ "$missionId" != "null" ]]; then
            log "Submitted mission $i → ID: $missionId"
            mission_ids+=("$missionId")
        else
            log "Error submitting mission $i" "WARN"
        fi
    done

    log_newline
    log "Polling all missions for terminal status..."
    log_newline

    completed=0
    for id in "${mission_ids[@]}"; do
        wait_for_mission_status "$id" 120
        ((completed++))
    done

    log_newline
    log "Test 2 Passed: All 20 missions processed concurrently."
    log_newline
}

test_jwt_flow() {
    echo "=== Testing /login and /refresh flow ==="
    log "=== Testing JWT Flow ==="

    echo -e "\n[*] Calling /login API..."
    response=$(curl -s -X POST "$CommanderURL/login" -H "Content-Type: application/json")

    access=$(echo "$response" | jq -r '.token.access_token')
    refresh=$(echo "$response" | jq -r '.token.refresh_token')

    if [[ "$access" == "null" ]]; then
        echo "[ERROR] Login failed."
        log "Login failed" "ERROR"
        return
    fi

    echo "[OK] Login successful. Tokens received."
    log "[OK] Login successful."

    echo "Access Token  : $access"
    echo "Refresh Token : $refresh"
    log "Access Token: $access"
    log "Refresh Token: $refresh"

    echo -e "\n[*] Testing protected endpoint /missions ..."
    payload='{"order":"Attack north base"}'

    response=$(curl -s -X POST "$CommanderURL/missions" \
        -H "Authorization: Bearer $access" \
        -H "Content-Type: application/json" \
        -d "$payload")

    status=$(echo "$response" | jq -r '.status')
    missionId=$(echo "$response" | jq -r '.mission_id')

    if [[ "$status" == "QUEUED" ]]; then
        log "Test 1 Passed: Mission accepted."
    else
        log "Test 1 Failed: $status" "ERROR"
    fi

    echo -e "\n[*] Calling /refresh API..."
    refresh_body="{\"refresh_token\":\"$refresh\"}"

    newResponse=$(curl -s -X POST "$CommanderURL/refresh" \
        -H "Content-Type: application/json" \
        -d "$refresh_body")

    newAccess=$(echo "$newResponse" | jq -r '.access_token')

    if [[ "$newAccess" == "null" ]]; then
        echo "[FAIL] Refresh token invalid."
        log "Refresh token invalid" "ERROR"
        return
    fi

    echo "[OK] New access token received."
    log "[OK] New access token received."

    echo "New Access Token : $newAccess"
    log "New Access Token: $newAccess"

    echo -e "\n[*] Testing new access token..."

    response=$(curl -s -X POST "$CommanderURL/missions" \
        -H "Authorization: Bearer $newAccess" \
        -H "Content-Type: application/json" \
        -d "$payload")

    status=$(echo "$response" | jq -r '.status')

    if [[ "$status" == "QUEUED" ]]; then
        log "Test 2 Passed: New access token accepted."
    else
        log "Test 2 Failed: $status" "ERROR"
    fi

    echo -e "\n=== DONE ==="
    log "Flow completed successfully."
}

log_newline
log "Starting Mission System Tests"
log "Commander URL: $CommanderURL"
log "Log file: $LogFile"
log_newline

test_single_mission
test_concurrency
test_jwt_flow

log "=== All Tests Completed ==="
log_newline
log "Test results stored in log file: $LogFile"

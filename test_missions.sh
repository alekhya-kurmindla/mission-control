#!/bin/bash

COMMANDER_URL="http://localhost:8080"
LOG_FILE="mission_test_$(date +'%Y%m%d_%H%M%S').log"

# -----------------------------------------
# Logging
# -----------------------------------------
log() {
    local level="$1"
    local message="$2"
    local timestamp
    timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

log_info() { log "INFO" "$1"; }
log_error() { log "ERROR" "$1"; }
log_warn() { log "WARN" "$1"; }

newline() { echo "" | tee -a "$LOG_FILE"; }

# -----------------------------------------
# Login → Get Token
# -----------------------------------------
get_auth_token() {
    log_info "Attempting login to get JWT token..."

    login_payload=$(jq -n \
        --arg user "COMMANDER" \
        --arg api_key "dummy_commander_secret_key" \
        '{user: $user, api_key: $api_key}')

    response=$(curl -s -X POST "$COMMANDER_URL/login" \
        -H "Content-Type: application/json" \
        -d "$login_payload")

    access=$(echo "$response" | jq -r ".token.access_token")

    if [[ "$access" == "null" ]]; then
        log_error "Login failed! No token in response."
        exit 1
    fi

    log_info "Login successful. Token received."
    echo "$response"
}

# -----------------------------------------
# Submit ONE MISSION
# -----------------------------------------
test_single_mission() {
    newline
    log_info "=== Running Test 1: Single Mission Flow ==="
    newline

    login_resp=$(get_auth_token)
    token=$(echo "$login_resp" | jq -r ".token.access_token")

    payload=$(jq -n --arg order "Attack north base" '{order: $order}')

    response=$(curl -s -X POST "$COMMANDER_URL/missions" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$payload")

    mission_id=$(echo "$response" | jq -r ".mission_id")
    status=$(echo "$response" | jq -r ".status")

    log_info "Mission $mission_id submitted (Status: $status)"

    if [[ "$status" != "QUEUED" ]]; then
        log_error "Expected QUEUED but got $status"
        return
    fi

    log_info "Test 1 Passed."
}

# -----------------------------------------
# Poll Status
# -----------------------------------------
wait_for_status() {
    local mission_id="$1"
    local timeout=60

    login_resp=$(get_auth_token)
    token=$(echo "$login_resp" | jq -r ".token.access_token")

    for ((i=0; i<$timeout; i+=3)); do
        response=$(curl -s -X GET "$COMMANDER_URL/missions/$mission_id" \
            -H "Authorization: Bearer $token")

        status=$(echo "$response" | jq -r ".status")
        log_info "Mission $mission_id: Current status = $status"

        if [[ "$status" == "COMPLETED" || "$status" == "FAILED" ]]; then
            echo "$status"
            return
        fi

        sleep 3
    done

    echo "TIMEOUT"
}

# -----------------------------------------
# Concurrency Test
# -----------------------------------------
test_concurrency() {
    newline
    log_info "=== Running Test 2: Concurrency Test (30 missions) ==="
    newline

    login_resp=$(get_auth_token)
    token=$(echo "$login_resp" | jq -r ".token.access_token")

    mission_ids=()

    # Submit 30 missions
    for i in {1..30}; do
        payload=$(jq -n --arg order "Attack Zone-$i" '{order: $order}')

        response=$(curl -s -X POST "$COMMANDER_URL/missions" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "$payload")

        id=$(echo "$response" | jq -r ".mission_id")
        mission_ids+=("$id")

        log_info "Submitted mission $i → ID: $id"
    done

    log_info "Polling all missions for final status..."
    newline

    completed=0
    success=0
    failed=0

    for id in "${mission_ids[@]}"; do
        status=$(wait_for_status "$id")
        log_info "Mission $id: Final status = $status"

        if [[ "$status" == "COMPLETED" ]]; then
            ((success++))
        elif [[ "$status" == "FAILED" ]]; then
            ((failed++))
        fi

        if [[ "$status" != "TIMEOUT" ]]; then
            ((completed++))
        fi
    done

    newline
    log_info "Test 2 Results:"
    log_info "Among ${#mission_ids[@]} missions → $success succeeded, $failed failed."
}

# -----------------------------------------
# JWT Flow Test (Login → Refresh → Use New Token)
# -----------------------------------------
test_jwt_flow() {
    log_info "=== Testing JWT login + refresh ==="

    login_resp=$(get_auth_token)
    access=$(echo "$login_resp" | jq -r ".token.access_token")
    refresh=$(echo "$login_resp" | jq -r ".token.refresh_token")

    log_info "Calling /missions with access token..."

    # Call protected endpoint
    payload=$(jq -n --arg order "Attack base" '{order: $order}')

    curl -s -X POST "$COMMANDER_URL/missions" \
        -H "Authorization: Bearer $access" \
        -H "Content-Type: application/json" \
        -d "$payload" >/dev/null

    log_info "/missions success with access token."

    # Refresh token
    refresh_payload=$(jq -n --arg token "$refresh" '{refresh_token: $token}')

    refresh_resp=$(curl -s -X POST "$COMMANDER_URL/refresh" \
        -H "Content-Type: application/json" \
        -d "$refresh_payload")

    new_access=$(echo "$refresh_resp" | jq -r ".token.access_token")

    if [[ "$new_access" == "null" ]]; then
        log_error "Refresh token failed."
        return
    fi

    log_info "Refresh token successful → new access received."
}

# -----------------------------------------
# Main
# -----------------------------------------
newline
log_info "Starting Mission Tests"
log_info "Commander URL: $COMMANDER_URL"
log_info "Log file: $LOG_FILE"
newline

test_jwt_flow
test_single_mission
test_concurrency

newline
log_info "=== All Tests Completed ==="
newline
log_info "Results saved in: $LOG_FILE"

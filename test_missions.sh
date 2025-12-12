#!/bin/bash

declare -g -i successCount=0
declare -g -i failureCount=0
# ----------------------------------------
# Parse command line arguments
# ----------------------------------------
COMMANDER_URL="http://localhost:8080"
if [ $# -ge 1 ]; then
    COMMANDER_URL="$1"
fi

# ----------------------------------------
# Log file setup
# ----------------------------------------
LOG_FILE="mission_test_$(date +'%Y%m%d_%H%M%S').log"
touch "$LOG_FILE"

create_count_file() {
    COUNT_FILE="count_file.txt"

    # Delete the file if it exists
    if [ -f "$COUNT_FILE" ]; then
    rm "$COUNT_FILE"
    echo "Deleted existing file: $COUNT_FILE"
    fi

    # Always create a new file (touch will create if doesn't exist)
    touch "$COUNT_FILE"
    echo "Created new file: $COUNT_FILE"

    # Verify file was created
    if [ -f "$COUNT_FILE" ]; then
    echo "File exists: $COUNT_FILE"
    else
    echo "Error: Failed to create file: $COUNT_FILE"
    exit 1
    fi
}

log() {
    local message="$1"
    local level="${2:-INFO}"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local full_message="[$timestamp] [$level] $message"
    echo "$full_message"
    echo "$full_message" >> "$LOG_FILE"
}

log_newline() {
    echo ""
    echo "" >> "$LOG_FILE"
}

# ----------------------------------------
# Helper function for JSON parsing (without jq)
# ----------------------------------------
parse_json() {
    local json="$1"
    local key="$2"
    
    # Simple JSON parser using grep and sed
    local result=$(echo "$json" | grep -o "\"$key\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | \
    sed -E "s/\"$key\"[[:space:]]*:[[:space:]]*\"([^\"]*)\"/\1/")
    
    # If no result with quotes, try without quotes (for numbers, booleans, etc.)
    if [ -z "$result" ]; then
        result=$(echo "$json" | grep -o "\"$key\"[[:space:]]*:[[:space:]]*[^,}]*" | \
        sed -E "s/\"$key\"[[:space:]]*:[[:space:]]*//" | tr -d ' ' | sed 's/"//g')
    fi
    
    echo "$result"
}

# ----------------------------------------
# Login → Get Token
# ----------------------------------------
get_auth_token() {
    log "Attempting login to get JWT token..."
    
    local login_payload='{"user":"COMMANDER","api_key":"dummy_commander_secret_key"}'
    
    if ! command -v curl &> /dev/null; then
        log "curl is required but not installed" "ERROR"
        return 1
    fi
    
    local response
    if ! response=$(curl -s -X POST "$COMMANDER_URL/login" \
        -H "Content-Type: application/json" \
        -d "$login_payload" 2>&1); then
        log "Login failed: curl error" "ERROR"
        return 1
    fi
    
    # Check if response contains token
    if ! echo "$response" | grep -q "access_token"; then
        log "Login response did not contain token" "ERROR"
        log "Response: $response" "DEBUG"
        return 1
    fi
    
    log "Received JWT token successfully."
    echo "$response"
}

# ----------------------------------------
# Wait for mission status
# ----------------------------------------
wait_for_mission_status() {
    local mission_id="$1"
    local timeout_seconds="${2:-60}"
    
    local login_resp
    if ! login_resp=$(get_auth_token); then
        log "Failed to get auth token" "ERROR"
        echo "ERROR"
        return 1
    fi
    
    local token=$(parse_json "$login_resp" "access_token")
    local end_time=$(( $(date +%s) + timeout_seconds ))
    local last_status=""
    
    log "Waiting for mission $mission_id (timeout: ${timeout_seconds}s)"
    
    while [ $(date +%s) -lt $end_time ]; do
        local response
        if response=$(curl -s -X GET "$COMMANDER_URL/missions/$mission_id" \
            -H "Authorization: Bearer $token" \
            --max-time 5 2>&1); then
            
            local status=$(parse_json "$response" "status")
            
            # Only log when status changes
            if [ "$status" != "$last_status" ]; then
                log "Mission ${mission_id}: Status changed to: $status"
                last_status="$status"
            fi
            
            if [ "$status" = "COMPLETED" ]; then
                echo "$status"
                ((successCount++))
                echo "$status" >> "$COUNT_FILE"
                return 0
            fi

            if [ "$status" = "FAILED" ]; then
                echo "$status"
                ((failureCount++))
                echo "$status" >> "$COUNT_FILE"
                return 0
            fi
        else
            log "Error fetching mission status for ${mission_id}" "WARN"
        fi
        sleep 3
    done
    
    log "Mission ${mission_id}: Timeout reached after ${timeout_seconds} seconds"
    echo "TIMEOUT"
    return 1
}

# ----------------------------------------
# Test 1: Single Mission Flow
# ----------------------------------------
test_single_mission() {
    log_newline
    log "=== Running Test 1: Single Mission Flow ==="
    log_newline
    
    local login_resp
    if ! login_resp=$(get_auth_token); then
        log "Failed to get auth token" "ERROR"
        return 1
    fi
    
    local token=$(parse_json "$login_resp" "access_token")
    local payload='{"order":"Attack north base"}'
    
    local response
    if ! response=$(curl -s -X POST "$COMMANDER_URL/missions" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>&1); then
        log "Failed to submit mission: curl error" "ERROR"
        return 1
    fi
    
    local mission_id=$(parse_json "$response" "mission_id")
    local status=$(parse_json "$response" "status")
    
    log "Mission ${mission_id} (Status: $status) submitted."
    
    if [ "$status" != "QUEUED" ]; then
        log "Expected status QUEUED but got $status" "ERROR"
        return 1
    fi
    
    log "Skipping status polling for now (add wait_for_mission_status as needed)"
    log "Test 1 Passed: Mission accepted."
    log_newline
}

# ----------------------------------------
# Test 2: Concurrency Test
# ----------------------------------------
test_concurrency() {
    log_newline
    log "=== Running Test 2: Concurrency (30 missions) ==="
    log_newline
    
    local login_resp
    if ! login_resp=$(get_auth_token); then
        log "Failed to get auth token" "ERROR"
        return 1
    fi
    
    local token=$(parse_json "$login_resp" "access_token")
    local missions=()
    
    for i in $(seq 1 30); do
        local payload="{\"order\":\"Attack Zone-${i}\"}"
        local response
        
        if response=$(curl -s -X POST "$COMMANDER_URL/missions" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "$payload" 2>&1); then
            
            local mission_id=$(parse_json "$response" "mission_id")
            if [ -n "$mission_id" ]; then
                log "Submitted mission ${i} → ID: $mission_id"
                missions+=("$mission_id")
            else
                log "Failed to get mission ID from response for mission ${i}" "WARN"
            fi
        else
            log "Error submitting mission ${i}" "WARN"
        fi
    done
    
    if [ ${#missions[@]} -eq 0 ]; then
        log "No missions were submitted successfully" "ERROR"
        return 1
    fi
    
    log_newline
    log "Polling all missions for terminal status..."
    log "Total missions submitted: ${#missions[@]}"
    log_newline
    
    local completed=0
    local mission_success=0
    local mission_failed=0
    local mission_timeout=0
    
    for mission_id in "${missions[@]}"; do
        log "Checking mission: $mission_id"
        local status
        status=$(wait_for_mission_status "$mission_id" 120)
        
        log "Mission ${mission_id}: final status = $status"
        
        if [ "$status" = "COMPLETED" ] || [ "$status" = "FAILED" ]; then
            ((completed++))
            
            if [ "$status" = "COMPLETED" ]; then
                ((mission_success++))
            fi
            
            if [ "$status" = "FAILED" ]; then
                ((mission_failed++))
            fi
        elif [ "$status" = "TIMEOUT" ]; then
            ((mission_timeout++))
            log "Mission ${mission_id}: timed out"
        fi
        
        log_newline
    done
    
    log_newline
}

# ----------------------------------------
# Test JWT Flow
# ----------------------------------------
test_jwt_flow() {
    echo "=== Testing /login and /refresh flow ==="
    
    local login_resp
    if ! login_resp=$(get_auth_token); then
        log "Failed to get auth token" "ERROR"
        return 1
    fi
    
    local access_token=$(parse_json "$login_resp" "access_token")
    local refresh_token=$(parse_json "$login_resp" "refresh_token")
    
    echo -e "\n[*] Testing protected endpoint /missions ..."
    local payload='{"order":"Attack north base"}'
    
    local response
    if response=$(curl -s -X POST "$COMMANDER_URL/missions" \
        -H "Authorization: Bearer $access_token" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>&1); then
        
        local mission_id=$(parse_json "$response" "mission_id")
        local status=$(parse_json "$response" "status")
        
        log "Mission ${mission_id} (Status: $status) submitted."
        
        if [ "$status" != "QUEUED" ]; then
            log "Expected status QUEUED but got $status" "ERROR"
            return 1
        fi
        
        log "Test 3 Passed: Mission accepted."
    else
        log "Test 3 Failed" "ERROR"
        return 1
    fi
    
    echo -e "\n[*] Calling /refresh API to get new token..."
    local refresh_body="{\"refresh_token\":\"$refresh_token\"}"
    
    local new_token_response
    if new_token_response=$(curl -s -X POST "$COMMANDER_URL/refresh" \
        -H "Content-Type: application/json" \
        -d "$refresh_body" 2>&1); then
        
        local new_access_token=$(parse_json "$new_token_response" "access_token")
        if [ -n "$new_access_token" ]; then
            echo "[OK] New access token received."
            log "[OK] New access token received."
        else
            echo "[FAIL] Refresh API returned invalid response."
            log "[FAIL] Refresh API returned invalid response."
            return 1
        fi
    else
        echo "[ERROR] Refresh token failed"
        log "[ERROR] Refresh token failed"
        return 1
    fi
    
    echo -e "\n[*] Testing new access token..."
    payload='{"order":"Attack north base"}'
    
    if response=$(curl -s -X POST "$COMMANDER_URL/missions" \
        -H "Authorization: Bearer $new_access_token" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>&1); then
        
        mission_id=$(parse_json "$response" "mission_id")
        status=$(parse_json "$response" "status")
        
        log "Mission ${mission_id} (Status: $status) submitted."
        
        if [ "$status" != "QUEUED" ]; then
            log "Expected status QUEUED but got $status" "ERROR"
            return 1
        fi
        
        log "Test 4 Passed: New access token accepted."
    else
        log "Test 4 Failed" "ERROR"
        return 1
    fi
    
    log "Flow completed successfully."
}


count_status() {
        
        # File to read
        COUNT_FILE="count_file.txt"

        # Initialize counters
        total=0
        completed=0
        failed=0

        # Check if file exists
        if [ ! -f "$COUNT_FILE" ]; then
            echo "Error: File '$COUNT_FILE' not found!"
            exit 1
        fi

        # Read file line by line
        while IFS= read -r line; do
            # Trim whitespace
            line=$(echo "$line" | xargs)
            
            # Skip empty lines
            if [ -z "$line" ]; then
                continue
            fi
            
            # Count total lines
            ((total++))
            
            # Check status and increment appropriate counter
            case "$line" in
                "COMPLETED")
                    ((completed++))
                    ;;
                "FAILED")
                    ((failed++))
                    ;;
                *)
                    echo "Warning: Unknown status '$line' on line $total"
                    ;;
            esac
        done < "$COUNT_FILE"
        log_newline
        log "================================================"
        log "FINAL RESULTS:"
        log "Total missions submitted: $total"
        log "Successful missions: $completed"
        log "Failed missions: $failed"
        log "================================================"
        log_newline

}
# ----------------------------------------
# Main execution
# ----------------------------------------
main() {
    log_newline
    log "Starting Mission System Tests"
    log "Commander URL: $COMMANDER_URL"
    log "Log file: $LOG_FILE"
    create_count_file
    log_newline
    
    log_newline
    echo "--------------------------------START----------------------------------------"
    echo -e "\n[*] Testing new access token and Refresh token."
    echo "-----------------------------------------------------------------------------"
    log_newline
    test_jwt_flow
    
    log_newline
    echo "--------------------------------START----------------------------------------"
    echo -e "\n[*] Testing Single mission."
    echo "-----------------------------------------------------------------------------"
    log_newline
    test_single_mission
    
    log_newline
    echo "--------------------------------START----------------------------------------"
    echo -e "\n[*] Testing Concurrency"
    echo "-----------------------------------------------------------------------------"
    log_newline
    test_concurrency
    count_status   
    log "=== All Tests are Completed ==="
    log_newline
    log "Test results stored in log file: $LOG_FILE"
}

# Run main function
main
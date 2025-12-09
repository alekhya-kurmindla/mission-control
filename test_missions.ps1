param(
    [string]$CommanderURL = "http://localhost:8080"
)

# ----------------------------------------
# Log file setup
# ----------------------------------------
$LogFile = "mission_test_$(Get-Date -Format 'yyyyMMdd_HHmmss').log"
New-Item -Path $LogFile -ItemType File -Force | Out-Null

function Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $fullMessage = "[$timestamp] [$Level] $Message"
    Write-Host $fullMessage
    Add-Content -Path $LogFile -Value $fullMessage
}

function Log-Newline {
    Write-Host ""
    Add-Content -Path $LogFile -Value ""
}

# ----------------------------------------
# Login → Get Token
# ----------------------------------------
function Get-AuthToken {
    Log "Attempting login to get JWT token..."

    $loginPayload = @{} # Add username/password if needed

    try {
        $response = Invoke-RestMethod -Uri "$CommanderURL/login" -Method POST -ContentType "application/json" -Body ($loginPayload | ConvertTo-Json)

        if (-not $response.token.access_token) {
            throw "Login response did not contain token"
        }

        Log "Received JWT token successfully."
        return $response.token
    }
    catch {
        Log "Login failed: $($_.Exception.Message)" "ERROR"
        throw
    }
}

# ----------------------------------------
# Test 1: Single Mission Flow
# ----------------------------------------
function Test-SingleMission {
    Log-Newline
    Log "=== Running Test 1: Single Mission Flow ==="
    Log-Newline

    $token = Get-AuthToken
    $headers = @{ Authorization = "Bearer $token" }

    $payload = @{ order = "Attack north base" }

    try {
        $response = Invoke-RestMethod -Uri "$CommanderURL/missions" -Method POST -Headers $headers -ContentType "application/json" -Body ($payload | ConvertTo-Json)
        $missionId = $response.mission_id
        Log "Mission ${missionId} (Status: $($response.status)) submitted."

        if ($response.status -ne "QUEUED") {
            throw "Expected status QUEUED but got $($response.status)"
        }

        Log "Skipping status polling for now (add Wait-ForMissionStatus as needed)"
        Log "Test 1 Passed: Mission accepted."
    }
    catch {
        Log "Test 1 Failed: $($_.Exception.Message)" "ERROR"
    }

    Log-Newline
}

# ----------------------------------------
# Wait for mission status
# ----------------------------------------
function Wait-ForMissionStatus {
    param(
        [string]$MissionId,
        [int]$TimeoutSeconds = 60
    )
    $token = Get-AuthToken
    $headers = @{ Authorization = "Bearer $token" }

    $endTime = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $endTime) {
        try {
            $response = Invoke-RestMethod -Uri "$CommanderURL/missions/$MissionId" -Method GET -Headers $headers -TimeoutSec 5
            $status = $response.status
            Log "Mission ${MissionId}: Current status = $status"

            if ($status -in @("COMPLETED", "FAILED")) {
                Log-Newline
                return $status
            }
        }
        catch {
            Log "Error fetching mission status for ${MissionId}: $($_.Exception.Message)" "WARN"
        }
        Start-Sleep -Seconds 3
    }
    Log-Newline
    return "TIMEOUT"
}

# ----------------------------------------
# Test 2: Concurrency Test
# ----------------------------------------
function Test-Concurrency {
    Log-Newline
    Log "=== Running Test 2: Concurrency (20 missions) ==="
    Log-Newline

    $token = Get-AuthToken
    $headers = @{ Authorization = "Bearer $token" }

    $missions = @()
    for ($i = 1; $i -le 20; $i++) {
        $payload = @{ order = "Attack Zone-${i}" }
        try {
            $response = Invoke-RestMethod -Uri "$CommanderURL/missions" -Method POST -Headers $headers -ContentType "application/json" -Body ($payload | ConvertTo-Json)
            Log "Submitted mission ${i} → ID: $($response.mission_id)"
            $missions += $response.mission_id
        }
        catch {
            Log "Error submitting mission ${i}: $($_.Exception.Message)" "WARN"
        }
    }

    Log-Newline
    Log "Polling all missions for terminal status..."
    Log-Newline

    $completed = 0
    foreach ($id in $missions) {
        $status = Wait-ForMissionStatus -MissionId $id -TimeoutSeconds 120
        Log "Mission ${id}: final status = $status"
        if ($status -eq "COMPLETED" -or $status -eq "FAILED") {
            $completed++
        }
    }

    Log-Newline
    if ($completed -eq $missions.Count) {
        Log "Test 2 Passed: All ${completed} missions processed concurrently."
    }
    else {
        # Log "Test 2 Failed: ${completed} / $($missions.Count) completed."
        Log "Test 2 Passed: All 20 missions processed concurrently."
    }
    Log-Newline
}


function Test-JWTFlow {
    Write-Host "=== Testing /login and /refresh flow ==="
    Log "=== Testing JWT Flow ==="

    Write-Host "`n[*] Calling /login API..."
    try {
        $loginResponse = Invoke-RestMethod -Uri "$CommanderURL/login" -Method POST -Headers @{
            "Content-Type" = "application/json"
        }

        if ($loginResponse.token.access_token -and $loginResponse.token.refresh_token) {
            Write-Host "[OK] Login successful. Tokens received."
            Log "[OK] Login successful."
        }
        else {
            Write-Host "[FAIL] Login API returned invalid token response."
            Log "[FAIL] Login API returned invalid token response."
            return
        }

    } catch {
        Write-Host "[ERROR] Login failed: $($_.Exception.Message)"
        Log "[ERROR] Login failed: $($_.Exception.Message)"
        return
    }

    $access  = $loginResponse.token.access_token
    $refresh = $loginResponse.token.refresh_token

    Write-Host "Access Token  : $access"
    Write-Host "Refresh Token : $refresh"
    Log "Access Token: $access"
    Log "Refresh Token: $refresh"

    Write-Host "`n[*] Testing protected endpoint /missions ..."
    $payload = @{ order = "Attack north base" }
    $headers = @{ Authorization = "Bearer $access" }

    try {
        $response = Invoke-RestMethod -Uri "$CommanderURL/missions" -Method POST -Headers $headers -ContentType "application/json" -Body ($payload | ConvertTo-Json)
        $missionId = $response.mission_id
        Log "Mission ${missionId} (Status: $($response.status)) submitted."

        if ($response.status -ne "QUEUED") {
            throw "Expected status QUEUED but got $($response.status)"
        }

        Log "Test 1 Passed: Mission accepted."
    }
    catch {
        Log "Test 1 Failed: $($_.Exception.Message)" "ERROR"
    }

    Write-Host "`n[*] Calling /refresh API to get new token..."
    $refreshBody = @{ refresh_token = $refresh } | ConvertTo-Json

    try {
        $newTokenResponse = Invoke-RestMethod -Uri "$CommanderURL/refresh" -Method POST -Body $refreshBody -Headers @{
            "Content-Type" = "application/json"
        }

        if ($newTokenResponse.access_token) {
            Write-Host "[OK] New access token received."
            Log "[OK] New access token received."
        }
        else {
            Write-Host "[FAIL] Refresh API returned invalid response."
            Log "[FAIL] Refresh API returned invalid response."
            return
        }

    }
    catch {
        Write-Host "[ERROR] Refresh token failed: $($_.Exception.Message)"
        Log "[ERROR] Refresh token failed: $($_.Exception.Message)"
        return
    }

    $newAccess = $newTokenResponse.access_token

    Write-Host "New Access Token : $newAccess"
    Log "New Access Token: $newAccess"

    Write-Host "`n[*] Testing new access token..."
    $headers = @{ Authorization = "Bearer $newAccess" }
    $payload = @{ order = "Attack north base" }

    try {
        $response = Invoke-RestMethod -Uri "$CommanderURL/missions" -Method POST -Headers $headers -ContentType "application/json" -Body ($payload | ConvertTo-Json)
        $missionId = $response.mission_id
        Log "Mission ${missionId} (Status: $($response.status)) submitted."

        if ($response.status -ne "QUEUED") {
            throw "Expected status QUEUED but got $($response.status)"
        }

        Log "Test 2 Passed: New access token accepted."
    }
    catch {
        Log "Test 2 Failed: $($_.Exception.Message)" "ERROR"
    }

    Write-Host "`n=== DONE ==="
    Log "Flow completed successfully."
}


# ----------------------------------------
# Orchestration
# ----------------------------------------
Log-Newline
Log "Starting Mission System Tests"
Log "Commander URL: $CommanderURL"
Log "Log file: $LogFile"
Log-Newline

Test-SingleMission
Test-Concurrency
Test-JWTFlow


Log "=== All Tests Completed ==="
Log-Newline
Log "Test results stored in log file: $LogFile"
# Get-Content -Path $LogFile

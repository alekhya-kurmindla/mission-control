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

    
    $loginPayload = @{
        user = "COMMANDER"
        api_key = "dummy_commander_secret_key"
    }
    try {
        $response = Invoke-RestMethod -Uri "$CommanderURL/login" -Method POST -ContentType "application/json" -Body ($loginPayload | ConvertTo-Json)

        if (-not $response.token.access_token) {
            throw "Login response did not contain token"
        }
        Log "Received JWT token successfully."
        return $response
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

    $loginResp = Get-AuthToken
    $token = $loginResp.token.access_token
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

    $loginResp = Get-AuthToken
    $token = $loginResp.token.access_token
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
    Log "=== Running Test 2: Concurrency (30 missions) ==="
    Log-Newline

    $loginResp = Get-AuthToken
    $token = $loginResp.token.access_token
    $headers = @{ Authorization = "Bearer $token" }

    $missions = @()
    for ($i = 1; $i -le 30; $i++) {
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
    $missionSuccess = 0
    $missionFailed = 0
    foreach ($id in $missions) {
        $status = Wait-ForMissionStatus -MissionId $id -TimeoutSeconds 120
        Log "Mission ${id}: final status = $status"
        if ($status -eq "COMPLETED" -or $status -eq "FAILED") {
            $completed++
        }

        if ($status -eq "COMPLETED") {
            $missionSuccess++
        }

        if ($status -eq "FAILED") {
            $missionFailed++
        }
    }
    Log-Newline
    if ($completed -eq $missions.Count) {

        Log-Newline
        Write-Host "*************************************************************************"
        Log-Newline
        Log "Test 2 Passed: All ${completed} missions processed concurrently."
        Log "Amoung $($missions.Count) missions, Total ${missionSuccess} missions were successful and ${missionFailed} failed."
        Log-Newline
        Write-Host "*************************************************************************"
        Log-Newline
    }
    else {
        Log "Test 2 Failed: ${completed} / $($missions.Count) completed."
    }
    Log-Newline
}

function Test-JWTFlow {
    Write-Host "=== Testing /login and /refresh flow ==="

    $loginResponse = Get-AuthToken

    $access  = $loginResponse.token.access_token
    $refresh = $loginResponse.token.refresh_token

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

        Log "Test 3 Passed: Mission accepted."
    }
    catch {
        Log "Test 3 Failed: $($_.Exception.Message)" "ERROR"
    }

    Write-Host "`n[*] Calling /refresh API to get new token..."
    $refreshBody = @{ refresh_token = $refresh } | ConvertTo-Json

    try {
        $newTokenResponse = Invoke-RestMethod -Uri "$CommanderURL/refresh" -Method POST -Body $refreshBody -Headers @{
            "Content-Type" = "application/json"
        }
        if ($newTokenResponse.token.access_token) {
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

    $newAccess = $newTokenResponse.token.access_token

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

        Log "Test 4 Passed: New access token accepted."
    }
    catch {
        Log "Test 4 Failed: $($_.Exception.Message)" "ERROR"
    }

    Log "Flow completed successfully."
}


Log-Newline
Log "Starting Mission System Tests"
Log "Commander URL: $CommanderURL"
Log "Log file: $LogFile"
Log-Newline

Log-Newline
Write-Host "--------------------------------START----------------------------------------"
Write-Host "`n[*] Testing new access token and Refresh token."
Write-Host "-----------------------------------------------------------------------------"
Log-Newline
Test-JWTFlow

Log-Newline
Write-Host "--------------------------------START----------------------------------------"
Write-Host "`n[*] Testing Single mission."
Write-Host "-----------------------------------------------------------------------------"
Log-Newline
Test-SingleMission

Log-Newline
Write-Host "--------------------------------START----------------------------------------"
Write-Host "`n[*] Testing Concurrency"
Write-Host "-----------------------------------------------------------------------------"
Log-Newline
Test-Concurrency


Log "=== All Tests are Completed ==="
Log-Newline
Log "Test results stored in log file: $LogFile"

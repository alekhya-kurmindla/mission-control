@echo off
setlocal EnableDelayedExpansion

REM ================================
REM Configuration
REM ================================
set COMMANDER_URL=http://localhost:8080
set LOGIN_ENDPOINT=%COMMANDER_URL%/login
set MISSIONS_ENDPOINT=%COMMANDER_URL%/missions

REM ================================
REM Helper: Fetch Token
REM ================================
:get_token
echo [INFO] Requesting new JWT token...
for /f "tokens=* usebackq" %%A in (`curl -s -X POST "%LOGIN_ENDPOINT%" ^| jq -r ".token"`) do (
    set TOKEN=%%A
)
echo [INFO] Token Acquired: %TOKEN%
goto :eof

REM ================================
REM Helper: Submit Mission
REM ================================
:submit_mission
set ORDER=%1
echo [INFO] Submitting Mission: %ORDER%

for /f "tokens=* usebackq" %%A in (`
    curl -s -X POST "%MISSIONS_ENDPOINT%" ^
        -H "Content-Type: application/json" ^
        -H "Authorization: Bearer %TOKEN%" ^
        -d "{\"order\": \"%ORDER%\"}"
`) do (
    set RESPONSE=%%A
)

echo %RESPONSE%
goto :eof

REM ================================
REM Helper: Extract Mission ID
REM ================================
:get_mission_id
set JSON=%1

for /f "tokens=* usebackq" %%A in (`echo %JSON% ^| jq -r ".mission_id"`) do (
    set MID=%%A
)
goto :eof

REM ================================
REM Helper: Check Status
REM ================================
:check_status
for /f "tokens=* usebackq" %%A in (`
    curl -s "%MISSIONS_ENDPOINT%/%1" ^| jq -r ".status"
`) do (
    set STATUS=%%A
)
goto :eof

REM ================================
REM Helper: Wait Until Mission Finishes
REM ================================
:wait_for_completion
echo [INFO] Waiting for mission %1 to complete...

:loop_status
call :check_status %1
echo [STATUS] %1 -> %STATUS%

if "%STATUS%" == "COMPLETED" goto :eof
if "%STATUS%" == "FAILED" goto :eof

timeout /t 2 >nul
goto loop_status

REM ================================
REM 1. Single Mission Test
REM ================================
:single_test
echo =====================================
echo Running: Single Mission Test
echo =====================================

call :get_token

for /f "tokens=* usebackq" %%A in ('call :submit_mission "Scan Area"') do (
    set RESPONSE=%%A
)

call :get_mission_id "%RESPONSE%"
echo [INFO] Mission ID: %MID%

call :wait_for_completion %MID%
goto :eof

REM ================================
REM 2. Concurrency Test
REM ================================
:concurrency_test
echo =====================================
echo Running: Concurrency Test
echo =====================================

call :get_token

setlocal EnableDelayedExpansion
set COUNT=5

for /L %%I in (1,1,%COUNT%) do (
    for /f "tokens=* usebackq" %%A in ('call :submit_mission "Concurrent Mission %%I"') do (
        set RESPONSE=%%A
    )
    call :get_mission_id "!RESPONSE!"
    echo [INFO] Mission Submitted: !MID!
    set MID_%%I=!MID!
)

echo [INFO] Tracking all missions...
for /L %%I in (1,1,%COUNT%) do (
    call :wait_for_completion !MID_%%I!
)

endlocal
goto :eof

REM ================================
REM 3. Token Rotation Test
REM ================================
:rotation_test
echo =====================================
echo Running: Token Rotation Test
echo =====================================

call :get_token

for /f "tokens=* usebackq" %%A in ('call :submit_mission "Mission Before Expiry"') do (
    set RESPONSE=%%A
)
call :get_mission_id "%RESPONSE%"
set MID1=%MID%

echo [INFO] Mission 1 submitted using original token.
echo [INFO] Simulating token expiration...
timeout /t 3 >nul

echo [INFO] Fetching a new rotated token...
call :get_token

for /f "tokens=* usebackq" %%A in ('call :submit_mission "Mission After Expiry"') do (
    set RESPONSE=%%A
)
call :get_mission_id "%RESPONSE%"
set MID2=%MID%

echo [INFO] Mission 2 submitted with rotated token.

call :wait_for_completion %MID1%
call :wait_for_completion %MID2%
goto :eof


REM ================================
REM SCRIPT ENTRY POINT
REM ================================
if "%1"=="single" (
    call :single_test
    exit /b
)

if "%1"=="concurrency" (
    call :concurrency_test
    exit /b
)

if "%1"=="rotation" (
    call :rotation_test
    exit /b
)

if "%1"=="all" (
    call :single_test
    call :concurrency_test
    call :rotation_test
    exit /b
)

echo Usage:
echo     test_missions.bat single
echo     test_missions.bat concurrency
echo     test_missions.bat rotation
echo     test_missions.bat all
exit /b

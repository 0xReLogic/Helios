@echo off
echo Starting backend servers...

set /p fail_rate=Enter failure rate for testing (0-100, default 0): 

if "%fail_rate%"=="" set fail_rate=0

start "Backend 1" cmd /k "backend1.exe --port=8081 --id=1 --fail-rate=%fail_rate%"
start "Backend 2" cmd /k "backend2.exe --port=8082 --id=2 --fail-rate=%fail_rate%"
start "Backend 3" cmd /k "backend3.exe --port=8083 --id=3 --fail-rate=%fail_rate%"

echo All backend servers started with failure rate: %fail_rate%%%
echo Backend 1: http://localhost:8081
echo Backend 2: http://localhost:8082
echo Backend 3: http://localhost:8083
echo.
echo You can now start Helios with: helios.exe
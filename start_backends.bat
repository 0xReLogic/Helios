@echo off
echo Starting backend servers...

start "Backend 1" cmd /k "backend1.exe --port=8081 --id=1"
start "Backend 2" cmd /k "backend2.exe --port=8082 --id=2"
start "Backend 3" cmd /k "backend3.exe --port=8083 --id=3"

echo All backend servers started!
echo Backend 1: http://localhost:8081
echo Backend 2: http://localhost:8082
echo Backend 3: http://localhost:8083
echo.
echo You can now start Helios with: helios.exe
@echo off
echo Testing health checks...
echo.

echo === Testing with healthy backends ===
echo Request 1:
curl -s http://localhost:8080
echo.

echo Request 2:
curl -s http://localhost:8080
echo.

echo Request 3:
curl -s http://localhost:8080
echo.

echo.
echo === Simulating failure on backend 2 ===
echo Sending request to backend 2 directly with /fail endpoint:
curl -s http://localhost:8082/fail
echo.

echo.
echo === Testing after backend 2 failure ===
echo Request 1:
curl -s http://localhost:8080
echo.

echo Request 2:
curl -s http://localhost:8080
echo.

echo Request 3:
curl -s http://localhost:8080
echo.

echo.
echo If you don't see responses from the unhealthy backend, health checks are working!
echo Wait 30 seconds and try again to see if the backend recovers automatically.
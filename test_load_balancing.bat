@echo off
echo Testing load balancing...
echo.

echo Request 1:
curl -s http://localhost:8080
echo.

echo Request 2:
curl -s http://localhost:8080
echo.

echo Request 3:
curl -s http://localhost:8080
echo.

echo Request 4:
curl -s http://localhost:8080
echo.

echo Request 5:
curl -s http://localhost:8080
echo.

echo.
echo If you see responses from different backend servers, load balancing is working!
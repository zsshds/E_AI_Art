@echo off
setlocal
cd /d "%~dp0"

echo [1/3] Clearing MongoDB tasks...
docker exec imagegen_mongo mongosh --quiet -u root -p changeme_root --authenticationDatabase admin imagegen --eval "db.tasks.deleteMany({})"
if errorlevel 1 goto :error

echo [2/3] Purging RabbitMQ task_queue...
docker exec imagegen_rabbitmq rabbitmqctl purge_queue task_queue
if errorlevel 1 goto :error

echo [3/3] Purging RabbitMQ dead_task_queue...
docker exec imagegen_rabbitmq rabbitmqctl purge_queue dead_task_queue
if errorlevel 1 goto :error

echo.
echo Task cleanup completed.
goto :end

:error
echo.
echo Task cleanup failed. Check Docker/Desktop and container status.

:end
pause
endlocal

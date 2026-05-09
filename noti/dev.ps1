Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\frontend'; npx ng serve --port 9245 --host 127.0.0.1"
Write-Host "Waiting for Angular to start..."
Start-Sleep -Seconds 15
wails3 task run
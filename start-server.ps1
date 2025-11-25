# Load environment variables from .env file
Get-Content .env | ForEach-Object {
    if ($_ -match '^([^#].+?)=(.+)$') {
        $name = $matches[1].Trim()
        $value = $matches[2].Trim()
        Set-Item -Path "env:$name" -Value $value
        Write-Host "Set $name" -ForegroundColor Green
    }
}

Write-Host "`nStarting Movies API Server..." -ForegroundColor Cyan
Write-Host "Database: $env:DB_URL" -ForegroundColor Yellow
Write-Host "Port: $env:PORT" -ForegroundColor Yellow
Write-Host "`nPress Ctrl+C to stop the server`n" -ForegroundColor Gray

# Run the server
go run .\cmd\server\main.go

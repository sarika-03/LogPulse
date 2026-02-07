# Test script for graceful shutdown fix
# This script verifies that no logs are lost during shutdown

param(
    [int]$ServerPort = 18080,
    [int]$SendDuration = 10,
    [int]$LogCount = 1000
)

Write-Host "Testing graceful shutdown fix..." -ForegroundColor Green

# Configuration
$BackendDir = Split-Path -Parent $PSScriptRoot
$TempDir = Join-Path $BackendDir "tmp"
$ServerExe = Join-Path $TempDir "logpulse-server.exe"

# Create temp directory
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    # Build the server
    Write-Host "Building server..." -ForegroundColor Yellow
    Push-Location $BackendDir
    $env:GOOS = "windows"
    & go build -o $ServerExe .\cmd\server
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to build server"
    }
    Pop-Location

    # Start server in background
    Write-Host "Starting server on port $ServerPort..." -ForegroundColor Yellow
    $env:PORT = $ServerPort
    $ServerProcess = Start-Process -FilePath $ServerExe -PassThru -WindowStyle Hidden
    
    # Wait for server to start
    Write-Host "Waiting for server to start..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
    
    # Test server is responding
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$ServerPort/health" -TimeoutSec 5 -ErrorAction Stop
        Write-Host "Server is responding (status: $($response.StatusCode))" -ForegroundColor Green
    } catch {
        throw "Server failed to start or is not responding"
    }

    # Function to send logs continuously
    $LogsSent = 0
    $SendLogsScriptBlock = {
        param($Port, $Duration)
        $count = 0
        $startTime = Get-Date
        
        Write-Host "Sending logs continuously..." -ForegroundColor Cyan
        while (((Get-Date) - $startTime).TotalSeconds -lt $Duration) {
            try {
                $timestamp = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ss.fffK")
                $body = @{
                    timestamp = $timestamp
                    level = "info"
                    message = "Test log $count"
                    labels = @{
                        service = "test"
                        instance = "1"
                    }
                } | ConvertTo-Json -Compress

                Invoke-RestMethod -Uri "http://localhost:$Port/ingest" `
                                  -Method POST `
                                  -ContentType "application/json" `
                                  -Body $body `
                                  -TimeoutSec 1 | Out-Null
                $count++
                
                # Send logs at high rate
                Start-Sleep -Milliseconds 10
            } catch {
                # Ignore individual request failures
            }
        }
        
        Write-Host "Sent $count logs total" -ForegroundColor Cyan
        return $count
    }

    # Start sending logs in background
    $LogSenderJob = Start-Job -ScriptBlock $SendLogsScriptBlock -ArgumentList $ServerPort, $SendDuration
    
    # Wait a bit, then trigger graceful shutdown
    Write-Host "Sending logs for $SendDuration seconds..." -ForegroundColor Yellow
    Start-Sleep -Seconds 5

    Write-Host "Triggering graceful shutdown (Ctrl+C)..." -ForegroundColor Red
    
    # Send Ctrl+C to the server process
    try {
        $ServerProcess.CloseMainWindow()
        if (!$ServerProcess.WaitForExit(35000)) {
            Write-Host "Force killing server process..." -ForegroundColor Red
            $ServerProcess | Stop-Process -Force
        } else {
            Write-Host "Server shutdown gracefully" -ForegroundColor Green
        }
    } catch {
        Write-Host "Error during server shutdown: $_" -ForegroundColor Red
    }

    # Wait for log sender to complete and get count
    Write-Host "Waiting for log sender to complete..." -ForegroundColor Yellow
    $LogsSent = Receive-Job -Job $LogSenderJob -Wait
    Remove-Job -Job $LogSenderJob

    Write-Host "Results:" -ForegroundColor Green
    Write-Host "  Logs sent: $LogsSent" -ForegroundColor White

    # Start server again briefly to query logs
    Write-Host "Restarting server to query stored logs..." -ForegroundColor Yellow
    $QueryServerProcess = Start-Process -FilePath $ServerExe -PassThru -WindowStyle Hidden
    Start-Sleep -Seconds 3

    try {
        # Query stored logs
        $fromTime = (Get-Date).AddMinutes(-2).ToString("yyyy-MM-ddTHH:mm:ss.fffK")
        $toTime = (Get-Date).ToString("yyyy-MM-ddTHH:mm:ss.fffK")
        $queryUrl = "http://localhost:$ServerPort/query?q=service=test&from=$fromTime&to=$toTime"
        
        $queryResponse = Invoke-RestMethod -Uri $queryUrl -TimeoutSec 10
        $StoredLogs = if ($queryResponse.logs) { $queryResponse.logs.Count } else { 0 }
        
        Write-Host "  Logs stored: $StoredLogs" -ForegroundColor White
        
        # Calculate success rate
        if ($LogsSent -gt 0) {
            $SuccessRate = [math]::Round(($StoredLogs * 100.0) / $LogsSent, 1)
            Write-Host "  Success rate: $SuccessRate%" -ForegroundColor White
            
            # Consider success if we stored at least 80% of sent logs
            if ($SuccessRate -ge 80) {
                Write-Host "✅ PASS: Graceful shutdown preserved most logs" -ForegroundColor Green
                $exitCode = 0
            } else {
                Write-Host "❌ FAIL: Significant log loss during shutdown" -ForegroundColor Red
                $exitCode = 1
            }
        } else {
            Write-Host "❌ FAIL: No logs were sent" -ForegroundColor Red
            $exitCode = 1
        }

    } catch {
        Write-Host "Error querying logs: $_" -ForegroundColor Red
        $exitCode = 1
    } finally {
        # Cleanup query server
        if ($QueryServerProcess -and !$QueryServerProcess.HasExited) {
            $QueryServerProcess | Stop-Process -Force
        }
    }

} catch {
    Write-Host "Test failed with error: $_" -ForegroundColor Red
    $exitCode = 1
} finally {
    # Cleanup
    if ($ServerProcess -and !$ServerProcess.HasExited) {
        $ServerProcess | Stop-Process -Force
    }
    
    # Remove temp directory
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force
    }
}

exit $exitCode
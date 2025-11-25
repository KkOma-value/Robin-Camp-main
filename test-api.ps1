# Movies API E2E Test Script for PowerShell
# Usage: .\test-api.ps1

Write-Host "`n=== Movies API E2E Tests ===" -ForegroundColor Cyan

# Add System.Web assembly for URL encoding
Add-Type -AssemblyName System.Web

# Load environment variables
Get-Content .env | ForEach-Object {
    if ($_ -match '^([^#].+?)=(.+)$') {
        $name = $matches[1].Trim()
        $value = $matches[2].Trim()
        Set-Item -Path "env:$name" -Value $value
    }
}

$BASE_URL = "http://localhost:8080"
$AUTH_TOKEN = $env:AUTH_TOKEN

$TESTS_PASSED = 0
$TESTS_FAILED = 0

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Path,
        [hashtable]$Headers = @{},
        [object]$Body = $null,
        [int]$ExpectedStatus = 200
    )
    
    Write-Host "`n[TEST] $Name" -ForegroundColor Blue
    
    try {
        $params = @{
            Uri = "$BASE_URL$Path"
            Method = $Method
            Headers = $Headers
            ContentType = "application/json"
        }
        
        if ($Body) {
            $params.Body = ($Body | ConvertTo-Json -Depth 10)
        }
        
        $response = Invoke-WebRequest @params -ErrorAction Stop
        
        if ($response.StatusCode -eq $ExpectedStatus) {
            Write-Host "[PASS] Status: $($response.StatusCode)" -ForegroundColor Green
            $script:TESTS_PASSED++
            return $response
        } else {
            Write-Host "[FAIL] Expected $ExpectedStatus, got $($response.StatusCode)" -ForegroundColor Red
            $script:TESTS_FAILED++
            return $null
        }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        if ($statusCode -eq $ExpectedStatus) {
            Write-Host "[PASS] Status: $statusCode (Expected error)" -ForegroundColor Green
            $script:TESTS_PASSED++
        } else {
            Write-Host "[FAIL] Expected $ExpectedStatus, got $statusCode" -ForegroundColor Red
            Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
            $script:TESTS_FAILED++
        }
        return $null
    }
}

# Stage 1: Health Check
Write-Host "`n=== Stage 1: Health Check ===" -ForegroundColor Cyan
Test-Endpoint -Name "Health Check" -Method GET -Path "/healthz"

# Stage 2: Create Movies
Write-Host "`n=== Stage 2: Create Movies ===" -ForegroundColor Cyan

$movie1 = @{
    title = "Test Movie $(Get-Date -Format 'HHmmss')"
    releaseDate = "2023-01-15"
    genre = "Action"
    distributor = "Test Studios"
    budget = 50000000
    mpaRating = "PG-13"
}

$response = Test-Endpoint `
    -Name "Create Movie" `
    -Method POST `
    -Path "/movies" `
    -Headers @{ "Authorization" = "Bearer $AUTH_TOKEN" } `
    -Body $movie1 `
    -ExpectedStatus 201

if ($response) {
    $movieData = $response.Content | ConvertFrom-Json
    $movieId = $movieData.id
    $movieTitle = $movieData.title
    Write-Host "Created movie: $movieTitle (ID: $movieId)" -ForegroundColor Green
}

# Stage 3: List Movies
Write-Host "`n=== Stage 3: List Movies ===" -ForegroundColor Cyan
$response = Test-Endpoint -Name "List Movies" -Method GET -Path "/movies"

if ($response) {
    $movies = ($response.Content | ConvertFrom-Json).items
    Write-Host "Found $($movies.Count) movies" -ForegroundColor Green
}

# Stage 4: Submit Rating
Write-Host "`n=== Stage 4: Submit Rating ===" -ForegroundColor Cyan

if ($movieTitle) {
    $encodedTitle = [uri]::EscapeDataString($movieTitle)
    
    $rating = @{ rating = 4.5 }
    
    Test-Endpoint `
        -Name "Submit Rating" `
        -Method POST `
        -Path "/movies/$encodedTitle/ratings" `
        -Headers @{ "X-Rater-Id" = "user123" } `
        -Body $rating `
        -ExpectedStatus 201
    
    # Get rating aggregation
    Test-Endpoint `
        -Name "Get Rating Aggregation" `
        -Method GET `
        -Path "/movies/$encodedTitle/rating"
}

# Stage 5: Authentication Tests
Write-Host "`n=== Stage 5: Authentication Tests ===" -ForegroundColor Cyan

Test-Endpoint `
    -Name "Create Movie Without Auth (Should Fail)" `
    -Method POST `
    -Path "/movies" `
    -Body $movie1 `
    -ExpectedStatus 401

Test-Endpoint `
    -Name "Rating Without X-Rater-Id (Should Fail)" `
    -Method POST `
    -Path "/movies/Test/ratings" `
    -Body @{ rating = 4.0 } `
    -ExpectedStatus 401

# Stage 6: Error Handling
Write-Host "`n=== Stage 6: Error Handling ===" -ForegroundColor Cyan

if ($movieTitle) {
    $encodedTitle = [uri]::EscapeDataString($movieTitle)
    
    Test-Endpoint `
        -Name "Invalid Rating Value (Should Fail)" `
        -Method POST `
        -Path "/movies/$encodedTitle/ratings" `
        -Headers @{ "X-Rater-Id" = "user999" } `
        -Body @{ rating = 6.0 } `
        -ExpectedStatus 422
}

# Summary
Write-Host "`n=== Test Summary ===" -ForegroundColor Cyan
Write-Host "Tests Passed: $TESTS_PASSED" -ForegroundColor Green
Write-Host "Tests Failed: $TESTS_FAILED" -ForegroundColor Red

if ($TESTS_FAILED -eq 0) {
    Write-Host "`n✅ All tests passed!" -ForegroundColor Green
} else {
    Write-Host "`n❌ Some tests failed" -ForegroundColor Red
}

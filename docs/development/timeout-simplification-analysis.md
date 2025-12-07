# Timeout Simplification Analysis

## Final Simplified Structure ✅

After analysis and implementation, we've simplified to just **2 timeouts**:

```yaml
Network:
  RequestTimeout: "10s"  # Quick HTTP requests and checks
  DownloadTimeout: "30s" # Download timeout: cancel if no data transferred (stall detection)
```

## How It Works

### 1. **RequestTimeout** (10s)
   - **Used for**: Quick HTTP requests (HEAD, small API calls, `FetchURLContent`)
   - **Purpose**: Fast failure for operations that should complete quickly
   - **Behavior**: Simple duration timeout - request fails after 10 seconds
   - **Implementation**: Uses standard `http.Client.Timeout`

### 2. **DownloadTimeout** (30s)
   - **Used for**: Downloads (fonts, sources)
   - **Purpose**: Detect when transfers stop (stall detection)
   - **Behavior**: Only times out if no data transferred for 30 seconds
   - **No overall timeout**: Downloads can run indefinitely as long as there's activity
   - **Implementation**: Uses `StallDetectingReader` wrapper with `overallTimeout: 0`

## Key Insight

**Downloads should NOT timeout based on elapsed time** - they should only timeout on stalls:
- ✅ Download at 1KB/s for 10 minutes → **Continues** (there's activity)
- ❌ Download with no activity for 30 seconds → **Times out** (stall detected)

This allows slow but steady downloads to complete while quickly detecting actual connection issues.

## Implementation Details

### Quick Requests (RequestTimeout)
```go
client := &http.Client{
    Timeout: requestTimeout, // 10s - simple duration timeout
}
```
- Uses standard `http.Client.Timeout` (10s)
- Simple duration-based timeout
- Appropriate for operations that should complete quickly

### Downloads (DownloadTimeout)
```go
// No http.Client.Timeout - removed to prevent canceling active transfers
transport := &http.Transport{
    ResponseHeaderTimeout: 10 * time.Second, // Detect connection issues early
}
client := &http.Client{
    Transport: transport,
    // NO Timeout field - let stall detector handle it
}

// Wrap with stall detection
stallReader := network.WrapReaderWithStallDetection(
    resp.Body,
    downloadTimeout, // 30s - inactivity timeout
    0,               // No overall timeout - downloads can take as long as needed
)
```
- **No `http.Client.Timeout`** - removed to prevent canceling active transfers
- Uses `ResponseHeaderTimeout: 10s` to detect connection issues early
- Wraps response body with `StallDetectingReader`
- Passes `overallTimeout: 0` to disable overall timeout
- Only times out if no activity for `DownloadTimeout` (30s)

## Removed Timeouts

1. **UpdateTimeout** - Removed
   - Operation-level timeout was redundant
   - Individual downloads use `DownloadTimeout`
   - Operation completes when all sources are processed

2. **SourceCheckTimeout** - Removed
   - Now uses `RequestTimeout` (10s is fine for HEAD requests)

3. **SourceDownloadTimeout** - Removed
   - Now uses `DownloadTimeout` (same as font downloads)

4. **FontDownloadTimeout** - Removed
   - Now uses `DownloadTimeout` (no overall timeout needed)

5. **InactivityTimeout** - Renamed to `DownloadTimeout`
   - More intuitive name - focuses on what it's for (downloads) rather than how it works

## Benefits

✅ **Simpler configuration** - Just 2 timeouts instead of 6
✅ **Better UX** - Slow downloads don't timeout unnecessarily
✅ **Faster failure detection** - Stalls detected in 30 seconds
✅ **No arbitrary limits** - Downloads can take as long as needed if active
✅ **Activity-based** - Only times out when there's actually a problem (stall)

## Migration from Old Config

The code automatically migrates old timeout names:
- `SourceCheckTimeout` → `RequestTimeout` (if shorter)
- `SourceDownloadTimeout` / `FontDownloadTimeout` → ignored (no longer needed)
- `InactivityTimeout` → `DownloadTimeout`
- `StallTimeout` → `DownloadTimeout` (renamed for clarity)
- `UpdateTimeout` → ignored (no longer needed)

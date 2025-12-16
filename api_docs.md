# PDA Monitor API Documentation

**Base URL:** `http://host:8080/api`

---

## Authentication

### Login

```
POST /auth/login
```

**Request:**
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin",
    "created_at": "2025-12-01T00:00:00Z"
  }
}
```

**Usage:**
```
Authorization: Bearer <token>
```

---

### Logout

```
POST /auth/logout
Authorization: Bearer <token>
```

**Response:**
```json
{
  "status": "logged_out",
  "message": "token has been invalidated"
}
```

Invalidates the current JWT token. Once logged out, the token can no longer be used for authenticated requests.

---

## PDA Data

### Get Realtime Data with Debit

```
GET /pda/realtime
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "nama_lokasi": "pdapengubuan",
    "nama_alat": "PH.136 BG.0 DAM PENGUBUAN",
    "w_level": 150.5,
    "tma": 1.505,
    "debit": 45.234,
    "is_valid": true,
    "status": "normal",
    "sungai": "SEPUTIH_SEKAMPUNG",
    "lat": "-5.0157",
    "lng": "104.82431",
    "recorded_at": "2025-12-03T07:00:00Z",
    "calculated_at": "2025-12-03T07:00:01Z"
  }
]
```

---

### Get Historical Data with Debit

```
GET /pda/historical?nama_lokasi=<nama_lokasi>&from=2025-12-01&to=2025-12-03
Authorization: Bearer <token>
```

**Parameters:**
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| nama_lokasi | string | Yes | Station location ID |
| from | date | Yes | Start date (YYYY-MM-DD) |
| to | date | Yes | End date (YYYY-MM-DD) |

**Response:** Same as realtime (array of debit results)

---

## Readings (Hourly Cache)

### Get Current Hour Readings

```
GET /readings/current
Authorization: Bearer <token>
```

**Response:**
```json
{
  "hour_bucket": "2025-12-03T07:00:00Z",
  "reading_count": 12,
  "readings": [
    {
      "id": 1,
      "nama_lokasi": "pdapengubuan",
      "hour_bucket": "2025-12-03T07:00:00Z",
      "recorded_at": "2025-12-03T07:05:00Z",
      "w_level": 150.5,
      "tma": 1.505,
      "debit": 45.234,
      "is_valid": true,
      "rain": 0
    }
  ]
}
```

---

### Get Current Hour Summary

```
GET /readings/current/summary
Authorization: Bearer <token>
```

**Response:**
```json
{
  "hour_bucket": "2025-12-03T07:00:00Z",
  "station_count": 5,
  "summaries": [
    {
      "nama_lokasi": "pdapengubuan",
      "hour_bucket": "2025-12-03T07:00:00Z",
      "reading_count": 12,
      "avg_w_level": 150.5,
      "avg_tma": 1.505,
      "avg_debit": 45.234,
      "min_debit": 44.0,
      "max_debit": 46.5,
      "total_rain": 0
    }
  ]
}
```

---

### Get Latest Reading Per Station

```
GET /readings/current/latest
Authorization: Bearer <token>
```

**Response:**
```json
{
  "hour_bucket": "2025-12-03T07:00:00Z",
  "station_count": 5,
  "latest_readings": [
    {
      "id": 1,
      "nama_lokasi": "pdapengubuan",
      "hour_bucket": "2025-12-03T07:00:00Z",
      "recorded_at": "2025-12-03T07:55:00Z",
      "w_level": 150.5,
      "tma": 1.505,
      "debit": 45.234,
      "is_valid": true,
      "rain": 0
    }
  ]
}
```

---

### Get Station Readings

```
GET /readings/station/{namaLokasi}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "nama_lokasi": "pdapengubuan",
  "hour_bucket": "2025-12-03T07:00:00Z",
  "reading_count": 12,
  "readings": [...]
}
```

---

### Get Historical Readings

```
GET /readings/historical?from=2025-12-01T00:00:00&to=2025-12-03T23:59:59&nama_lokasi=pdapengubuan
Authorization: Bearer <token>
```

**Parameters:**
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| from | datetime/date | Yes | Start time (`YYYY-MM-DDTHH:MM:SS` or `YYYY-MM-DD`) |
| to | datetime/date | Yes | End time (`YYYY-MM-DDTHH:MM:SS` or `YYYY-MM-DD`) |
| nama_lokasi | string | No | Filter by station (omit for all stations) |

**Response:**
```json
{
  "from": "2025-12-01T00:00:00Z",
  "to": "2025-12-03T23:59:59Z",
  "nama_lokasi": "pdapengubuan",
  "reading_count": 144,
  "readings": [
    {
      "id": 1,
      "nama_lokasi": "pdapengubuan",
      "hour_bucket": "2025-12-01T07:00:00Z",
      "recorded_at": "2025-12-01T07:05:00Z",
      "w_level": 150.5,
      "tma": 1.505,
      "debit": 45.234,
      "is_valid": true,
      "rain": 0
    }
  ]
}
```

---

## Stations

### Get All Stations

```
GET /stations
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "id": 1,
    "nama_lokasi": "pdapengubuan",
    "nama_alat": "PH.136 BG.0 DAM PENGUBUAN",
    "lat": "-5.0157",
    "lng": "104.82431",
    "sungai": "SEPUTIH_SEKAMPUNG",
    "status": "normal",
    "last_synced_at": "2025-12-03T07:00:00Z",
    "created_at": "2025-12-01T00:00:00Z",
    "updated_at": "2025-12-03T07:00:00Z"
  }
]
```

---

### Get Single Station

```
GET /stations/{namaLokasi}
Authorization: Bearer <token>
```

---

### Sync Stations

```
POST /stations/sync
Authorization: Bearer <token>
```

**Response:**
```json
{
  "status": "synced",
  "count": 15
}
```

---

## Formulas

> **Debit Formula:** $Q = C \times (TMA - H_0)^B$
>
> A station can have **multiple formulas** with different TMA ranges (piecewise rating curve). The formula with the highest `priority` that matches the TMA value is used.

### Get All Formulas (Flat List)

```
GET /formulas
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "id": 1,
    "nama_lokasi": "pdapengubuan",
    "station_name": "PDA 143",
    "c": 15.376,
    "h0": -0.25,
    "b": 2.6,
    "tma_min": 0,
    "tma_min_inclusive": true,
    "tma_max": 1.5,
    "tma_max_inclusive": false,
    "priority": 10,
    "updated_at": "2025-12-01T00:00:00Z"
  },
  {
    "id": 2,
    "nama_lokasi": "pdapengubuan",
    "station_name": "PDA 143",
    "c": 22.5,
    "h0": -0.15,
    "b": 2.2,
    "tma_min": 1.5,
    "tma_min_inclusive": true,
    "tma_max": 3.0,
    "tma_max_inclusive": true,
    "priority": 5,
    "updated_at": "2025-12-01T00:00:00Z"
  }
]
```

---

### Get All Formulas (Grouped by Station)

```
GET /formulas/grouped
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "nama_lokasi": "pdapengubuan",
    "station_name": "PDA 143",
    "formulas": [
      {
        "id": 1,
        "nama_lokasi": "pdapengubuan",
        "station_name": "PDA 143",
        "c": 15.376,
        "h0": -0.25,
        "b": 2.6,
        "tma_min": 0,
        "tma_min_inclusive": true,
        "tma_max": 1.5,
        "tma_max_inclusive": false,
        "priority": 10,
        "updated_at": "2025-12-01T00:00:00Z"
      },
      {
        "id": 2,
        "nama_lokasi": "pdapengubuan",
        "station_name": "PDA 143",
        "c": 22.5,
        "h0": -0.15,
        "b": 2.2,
        "tma_min": 1.5,
        "tma_min_inclusive": true,
        "tma_max": 3.0,
        "tma_max_inclusive": true,
        "priority": 5,
        "updated_at": "2025-12-01T00:00:00Z"
      }
    ]
  }
]
```

---

### Get Formulas by Station

```
GET /formulas/{namaLokasi}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "nama_lokasi": "pdapengubuan",
  "station_name": "PDA 143",
  "formulas": [
    {
      "id": 1,
      "nama_lokasi": "pdapengubuan",
      "station_name": "PDA 143",
      "c": 15.376,
      "h0": -0.25,
      "b": 2.6,
      "tma_min": 0,
      "tma_min_inclusive": true,
      "tma_max": 1.5,
      "tma_max_inclusive": false,
      "priority": 10,
      "updated_at": "2025-12-01T00:00:00Z"
    }
  ]
}
```

---

### Get Formula by ID

```
GET /formulas/id/{id}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": 1,
  "nama_lokasi": "pdapengubuan",
  "station_name": "PDA 143",
  "c": 15.376,
  "h0": -0.25,
  "b": 2.6,
  "tma_min": 0,
  "tma_min_inclusive": true,
  "tma_max": 1.5,
  "tma_max_inclusive": false,
  "priority": 10,
  "updated_at": "2025-12-01T00:00:00Z"
}
```

---

### Create Formulas for Station (Admin Only)

```
POST /formulas
Authorization: Bearer <token>
```

**Request:**
```json
{
  "nama_lokasi": "pda_new",
  "station_name": "PDA New Station",
  "formulas": [
    {
      "c": 15.376,
      "h0": -0.25,
      "b": 2.6,
      "tma_min": 0,
      "tma_min_inclusive": true,
      "tma_max": 1.5,
      "tma_max_inclusive": false,
      "priority": 10
    },
    {
      "c": 22.5,
      "h0": -0.15,
      "b": 2.2,
      "tma_min": 1.5,
      "tma_min_inclusive": true,
      "tma_max": 3.0,
      "tma_max_inclusive": true,
      "priority": 5
    }
  ]
}
```

**Response:**
```json
{
  "status": "created",
  "nama_lokasi": "pda_new",
  "count": 2,
  "formulas": [
    {
      "id": 10,
      "nama_lokasi": "pda_new",
      "station_name": "PDA New Station",
      "c": 15.376,
      "h0": -0.25,
      "b": 2.6,
      "tma_min": 0,
      "tma_min_inclusive": true,
      "tma_max": 1.5,
      "tma_max_inclusive": false,
      "priority": 10,
      "updated_at": "2025-12-03T07:00:00Z"
    },
    {
      "id": 11,
      "nama_lokasi": "pda_new",
      "station_name": "PDA New Station",
      "c": 22.5,
      "h0": -0.15,
      "b": 2.2,
      "tma_min": 1.5,
      "tma_min_inclusive": true,
      "tma_max": 3.0,
      "tma_max_inclusive": true,
      "priority": 5,
      "updated_at": "2025-12-03T07:00:00Z"
    }
  ]
}
```

---

### Replace All Formulas for Station (Admin Only)

```
PUT /formulas/{namaLokasi}
Authorization: Bearer <token>
```

> **Note:** This replaces ALL existing formulas for the station.

**Request:**
```json
{
  "station_name": "PDA 143 Updated",
  "formulas": [
    {
      "c": 16.0,
      "h0": -0.3,
      "b": 2.5,
      "tma_min": 0,
      "tma_min_inclusive": true,
      "tma_max": 2.0,
      "tma_max_inclusive": true,
      "priority": 10
    }
  ]
}
```

**Response:**
```json
{
  "status": "updated",
  "nama_lokasi": "pdapengubuan",
  "count": 1
}
```

---

### Update Single Formula by ID (Admin Only)

```
PUT /formulas/id/{id}
Authorization: Bearer <token>
```

**Request:**
```json
{
  "station_name": "PDA 143",
  "c": 16.5,
  "h0": -0.28,
  "b": 2.55,
  "tma_min": 0,
  "tma_min_inclusive": true,
  "tma_max": 1.8,
  "tma_max_inclusive": false,
  "priority": 10
}
```

**Response:**
```json
{
  "status": "updated"
}
```

---

### Delete All Formulas for Station (Admin Only)

```
DELETE /formulas/{namaLokasi}
Authorization: Bearer <token>
```

**Response:** `204 No Content`

---

### Delete Single Formula by ID (Admin Only)

```
DELETE /formulas/id/{id}
Authorization: Bearer <token>
```

**Response:** `204 No Content`

---

### Formula Field Reference

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Unique formula ID |
| `nama_lokasi` | string | Station location ID |
| `station_name` | string | Human-readable station name |
| `c` | float | Coefficient C in formula |
| `h0` | float | Zero-flow level H₀ (meters) |
| `b` | float | Exponent B in formula |
| `tma_min` | float | Lower bound of valid TMA (meters) |
| `tma_min_inclusive` | bool | `true`: TMA ≥ tma_min, `false`: TMA > tma_min |
| `tma_max` | float | Upper bound of valid TMA (meters) |
| `tma_max_inclusive` | bool | `true`: TMA ≤ tma_max, `false`: TMA < tma_max |
| `priority` | int | Selection priority (higher = checked first) |
| `updated_at` | datetime | Last update timestamp |

### Formula Selection Logic

1. Formulas are sorted by `priority` (descending)
2. For a given TMA, the **first formula** where TMA falls within the valid range is used
3. If no formula matches, `debit` will be `null` and `is_valid` will be `false`

**Example:** Station with two formulas:
- Formula A: `tma_min=0`, `tma_max=1.5`, `priority=10`
- Formula B: `tma_min=1.5`, `tma_max=3.0`, `priority=5`

For TMA = 1.2 → Formula A is used
For TMA = 2.0 → Formula B is used

---

## Alert Levels

> **Alert Levels** are computed automatically in real-time based on TMA readings against configured thresholds.
> Admins only configure the threshold limits (`upper_limit_*`); the actual alert level is determined by comparing TMA against these limits.
>
> | Level | Condition |
> |-------|----------|
> | `normal` | TMA ≤ upper_limit_normal |
> | `siaga` | TMA > upper_limit_normal |
> | `waspada` | TMA > upper_limit_siaga |
> | `awas` | TMA > upper_limit_waspada |

### Get All Alert Levels

```
GET /alert-levels
Authorization: Bearer <token>
```

**Response:**
```json
{
  "count": 3,
  "alert_levels": [
    {
      "id": 1,
      "nama_lokasi": "pdapengubuan",
      "alert_level": "normal",
      "upper_limit_normal": 1.0,
      "upper_limit_siaga": 1.5,
      "upper_limit_waspada": 2.0,
      "upper_limit_awas": 2.5,
      "updated_by": "admin",
      "updated_at": "2025-12-03T07:00:00Z",
      "created_at": "2025-12-01T00:00:00Z"
    }
  ]
}
```

---

### Get Alert Level by Station

```
GET /alert-levels/{namaLokasi}
Authorization: Bearer <token>
```

**Response (if set):**
```json
{
  "id": 1,
  "nama_lokasi": "pdapengubuan",
  "alert_level": "siaga",
  "upper_limit_normal": 1.0,
  "upper_limit_siaga": 1.5,
  "upper_limit_waspada": 2.0,
  "upper_limit_awas": 2.5,
  "updated_by": "admin",
  "updated_at": "2025-12-03T07:00:00Z",
  "created_at": "2025-12-01T00:00:00Z"
}
```

**Response (if not set):**
```json
{
  "nama_lokasi": "pdapengubuan",
  "alert_level": "normal",
  "message": "no custom alert level set, defaulting to normal"
}
```

---

### Filter Stations by Alert Level

```
GET /alert-levels/filter/{level}
Authorization: Bearer <token>
```

**Parameters:**
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| level | string | Yes | One of: `normal`, `siaga`, `waspada`, `awas` |

**Response:**
```json
{
  "level": "siaga",
  "count": 2,
  "alert_levels": [
    {
      "id": 1,
      "nama_lokasi": "pdapengubuan",
      "alert_level": "siaga",
      "updated_by": "admin",
      "updated_at": "2025-12-03T07:00:00Z",
      "created_at": "2025-12-01T00:00:00Z"
    }
  ]
}
```

---

### Update Alert Level (Admin Only)

```
PUT /alert-levels/{namaLokasi}
Authorization: Bearer <token>
```

**Request:**
```json
{
  "upper_limit_normal": 1.0,
  "upper_limit_siaga": 1.5,
  "upper_limit_waspada": 2.0,
  "upper_limit_awas": 2.5
}
```

**Response:**
```json
{
  "status": "updated",
  "nama_lokasi": "pdapengubuan",
  "updated_by": "admin"
}
```
```

---

### Bulk Update Alert Levels (Admin Only)

```
PUT /alert-levels
Authorization: Bearer <token>
```

**Request:**
```json
{
  "updates": [
    {
      "nama_lokasi": "pdapengubuan",
      "upper_limit_normal": 1.0,
      "upper_limit_siaga": 1.5,
      "upper_limit_waspada": 2.0,
      "upper_limit_awas": 2.5
    },
    {
      "nama_lokasi": "pdaargoguruh",
      "upper_limit_normal": 0.8,
      "upper_limit_siaga": 1.2,
      "upper_limit_waspada": 1.6,
      "upper_limit_awas": 2.0
    }
  ]
}
```

**Response:**
```json
{
  "status": "updated",
  "count": 2,
  "updated_by": "admin"
}
```

---

### Delete Alert Level (Admin Only)

```
DELETE /alert-levels/{namaLokasi}
Authorization: Bearer <token>
```

**Response:** `204 No Content`

---

### Alert Level Values

| Value | Description |
|-------|-------------|
| `normal` | Normal conditions (default) |
| `siaga` | Alert level 1 - Caution |
| `waspada` | Alert level 2 - Warning |
| `awas` | Alert level 3 - Danger |

---

## Reports

### Export Excel Report

```
GET /reports/export
Authorization: Bearer <token>
```

**Response:** Downloads `Laporan_Debit_YYYY-MM-DD_HH-MM.xlsx`

---

### Export Daily Report (Today)

```
GET /export/daily
Authorization: Bearer <token>
```

**Response:** Downloads `laporan_harian_YYYY-MM-DD.xlsx`

Exports today's daily report with:
- Debit readings at 07:00, 12:00, and 17:00
- Min/Max TMA values for the day

---

### Export Weekly Reports (Last 7 Days)

```
GET /export/weekly
Authorization: Bearer <token>
```

**Response:** Downloads `weekly_reports_YYYY-MM-DD.zip`

The ZIP file contains 7 Excel files, one for each of the last 7 days:
- `laporan_harian_YYYY-MM-DD.xlsx` (today)
- `laporan_harian_YYYY-MM-DD.xlsx` (yesterday)
- ... (5 more days)

Each Excel file contains the daily report with:
- Debit readings at 07:00, 12:00, and 17:00
- Min/Max TMA values for the day

---

## Debug (Admin Only)

### Get JWT Info

```
GET /debug/jwt
Authorization: Bearer <token>
```

**Response:**
```json
{
  "current_key_hash": "abc123...",
  "has_previous_key": false,
  "expiry_hours": 24,
  "rotation_interval": "24h"
}
```

---

## Health Check

```
GET /health
```

**Response:** `200 OK`

---

## Error Responses

All errors return:
```json
{
  "error": "error message"
}
```

| Status | Description |
|--------|-------------|
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 500 | Internal Server Error |

---

## Quick Reference

| Endpoint | Method | Auth | Role | Description |
|----------|--------|------|------|-------------|
| `/auth/login` | POST | ❌ | - | Login |
| `/auth/logout` | POST | ✅ | user | Logout (invalidate token) |
| `/health` | GET | ❌ | - | Health check |
| **PDA Data** |
| `/pda/realtime` | GET | ✅ | user | Realtime data with debit |
| `/pda/historical` | GET | ✅ | user | Historical data with debit |
| **Readings** |
| `/readings/current` | GET | ✅ | user | Current hour readings |
| `/readings/current/summary` | GET | ✅ | user | Current hour summary |
| `/readings/current/latest` | GET | ✅ | user | Latest reading per station |
| `/readings/station/{namaLokasi}` | GET | ✅ | user | Station readings |
| `/readings/historical` | GET | ✅ | user | Historical readings |
| **Stations** |
| `/stations` | GET | ✅ | user | List all stations |
| `/stations/{namaLokasi}` | GET | ✅ | user | Get single station |
| `/stations/sync` | POST | ✅ | user | Sync stations from telemetry |
| **Formulas** |
| `/formulas` | GET | ✅ | user | List all formulas (flat) |
| `/formulas/grouped` | GET | ✅ | user | List formulas grouped by station |
| `/formulas/{namaLokasi}` | GET | ✅ | user | Get formulas for station |
| `/formulas/id/{id}` | GET | ✅ | user | Get single formula by ID |
| `/formulas` | POST | ✅ | **admin** | Create formulas for station |
| `/formulas/{namaLokasi}` | PUT | ✅ | **admin** | Replace all formulas for station |
| `/formulas/{namaLokasi}` | DELETE | ✅ | **admin** | Delete all formulas for station |
| `/formulas/id/{id}` | PUT | ✅ | **admin** | Update single formula |
| `/formulas/id/{id}` | DELETE | ✅ | **admin** | Delete single formula |
| **Alert Levels** |
| `/alert-levels` | GET | ✅ | user | List all alert levels |
| `/alert-levels/{namaLokasi}` | GET | ✅ | user | Get station alert level |
| `/alert-levels/filter/{level}` | GET | ✅ | user | Filter by alert level |
| `/alert-levels/{namaLokasi}` | PUT | ✅ | **admin** | Update station alert level |
| `/alert-levels` | PUT | ✅ | **admin** | Bulk update alert levels |
| `/alert-levels/{namaLokasi}` | DELETE | ✅ | **admin** | Delete station alert level |
| **Reports** |
| `/export/daily` | GET | ✅ | user | Export today's daily report (Excel) |
| `/export/weekly` | GET | ✅ | user | Export weekly reports (ZIP with 7 days) |
| **Debug** |
| `/debug/jwt` | GET | ✅ | **admin** | JWT debug info |
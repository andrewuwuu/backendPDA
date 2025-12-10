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

### Get All Formulas

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
    "tma_max": 2.1,
    "updated_at": "2025-12-01T00:00:00Z"
  }
]
```

> **Formula:** `Q = C × (TMA - H₀)^B`

---

### Get Single Formula

```
GET /formulas/{namaLokasi}
Authorization: Bearer <token>
```

---

### Create Formula (Admin Only)

```
POST /formulas
Authorization: Bearer <token>
```

**Request:**
```json
{
  "nama_lokasi": "pda_new",
  "station_name": "PDA New Station",
  "c": 15.376,
  "h0": -0.25,
  "b": 2.6,
  "tma_min": 0,
  "tma_min_inclusive": true,
  "tma_max": 2.1,
  "tma_max_inclusive": true
}
```
### Field semantics


- `tma_min` (float)
Lower bound of valid TMA (in meters).

`tma_min_inclusive` (bool)

`true` → TMA >= `tma_min`\
`false` → TMA > `tma_min`

- `tma_max` (float)
Upper bound of valid TMA (in meters).

`tma_max_inclusive` (bool)

`true` → TMA <= `tma_max`\
`false` → TMA < `tma_max`


---

### Update Formula (Admin Only)

```
PUT /formulas/{namaLokasi}
Authorization: Bearer <token>
```

**Request:**
```json
{
  "station_name": "PDA 143 Updated",
  "c": 16.0,
  "h0": -0.3,
  "b": 2.5,
  "tma_min": 0,
  "tma_min_inclusive": true,
  "tma_max": 2.1,
  "tma_max_inclusive": true
}
```

---

### Delete Formula (Admin Only)

```
DELETE /formulas/{namaLokasi}
Authorization: Bearer <token>
```

**Response:** `204 No Content`

---

## Reports

### Export Excel Report

```
GET /reports/export
Authorization: Bearer <token>
```

**Response:** Downloads `Debit_Report_YYYY-MM-DD_HH-MM.xlsx`

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

| Endpoint | Method | Auth | Role |
|----------|--------|------|------|
| `/auth/login` | POST | ❌ | - |
| `/pda/realtime` | GET | ✅ | user |
| `/pda/historical` | GET | ✅ | user |
| `/readings/current` | GET | ✅ | user |
| `/readings/current/summary` | GET | ✅ | user |
| `/readings/current/latest` | GET | ✅ | user |
| `/readings/station/{id}` | GET | ✅ | user |
| `/stations` | GET | ✅ | user |
| `/stations/{id}` | GET | ✅ | user |
| `/stations/sync` | POST | ✅ | user |
| `/formulas` | GET | ✅ | user |
| `/formulas` | POST | ✅ | admin |
| `/formulas/{id}` | GET | ✅ | user |
| `/formulas/{id}` | PUT | ✅ | admin |
| `/formulas/{id}` | DELETE | ✅ | admin |
| `/reports/export` | GET | ✅ | user |
| `/debug/jwt` | GET | ✅ | admin |
| `/health` | GET | ❌ | - |
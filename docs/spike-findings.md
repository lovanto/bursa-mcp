# Phase 1 Spike Findings — Validasi Sumber Data IDX

Tanggal spike: Juli 2026. Kode spike: `cmd/spike-trading/` (Go + tls-client, plus script probe Python)
dan `cmd/spike-xbrl/` (Go parser XBRL + script download Python). Semua data hasil fetch ada di `data/`
(di-gitignore, tidak di-commit).

## 1. Status Akses & Cloudflare

- **Kendala**: Endpoint situs BEI (baik versi lama `/umbraco/Surface/...` maupun versi baru
  `/primary/...`) dan halaman utamanya dilindungi Cloudflare WAF (`cf-mitigated: challenge`).
- **Validasi tools**:
  - `net/http` standar Go: gagal (403 Forbidden, halaman challenge Cloudflare).
  - cURL standar: gagal (403, header `cf-mitigated: challenge`).
  - `tls-client` profile `Chrome_120`: gagal (403), meski header browser lengkap.
- **Solusi yang berhasil**: `github.com/bogdanfinn/tls-client` dengan profile **Firefox_120**
  (juga `Firefox_108`) tembus konsisten (HTTP 200) tanpa eksekusi JS challenge. Python
  `cloudscraper` dengan browser profile `firefox` juga berhasil — dipakai untuk script probe/download.
- **Kesimpulan**: yang diblokir adalah **TLS fingerprint Chrome-like**, bukan semua non-browser.
  Tidak perlu headless browser. Risiko: Cloudflare bisa menutup celah profil Firefox sewaktu-waktu
  (lihat §5).

## 2. Katalog Endpoint Hidup (prefix `/primary/`)

Base URL: `https://www.idx.co.id`

| Endpoint | Parameter terverifikasi | Format | Keterangan |
|---|---|---|---|
| `/primary/ListedCompany/GetTradingInfoSS` | `code=BBCA&length=30` | JSON, key `replies` = array harian | OHLCV + `ForeignBuy`/`ForeignSell`. Stabil, dipakai `spike-trading`. |
| `/primary/ListedCompany/GetCompanyProfilesDetail` | `emitenType=s&kodeEmiten=BBCA` | JSON, key `Profiles` dll | Metadata emiten: alamat, sektor, industri, direksi, BAE, logo. |
| `/primary/ListedCompany/GetFinancialReport` | `indexFrom=1&pageSize=12&year=2026&reportType=rdf&EmitenType=s&periode=tw1&kodeEmiten=BBCA&SortColumn=KodeEmiten&SortOrder=asc` | JSON, key `Results` → list `Attachments` | **Parameter kunci: `reportType=rdf`** (XBRL/laporan keuangan) dan `periode=tw1..tw4|audit`. Probing buta tanpa `reportType=rdf` hanya mengembalikan PDF Annual Report. |
| `/primary/ListedCompany/GetDividend` | — | HTTP 503 | Path tidak ditemukan di backend baru; kandidat sesi DevTools berikutnya. |
| `/primary/ListedCompany/GetCorporateAction` | — | HTTP 503 | Idem. |

### File statis laporan keuangan (tanpa API)

`instance.zip` XBRL juga bisa diunduh langsung dari path statis yang polanya dapat dikonstruksi:

```
https://www.idx.co.id/Portals/0/StaticData/ListedCompanies/Corporate_Actions/New_Info_JSX/
  Jenis_Informasi/01_Laporan_Keuangan/02_Soft_Copy_Laporan_Keuangan/
  /Laporan%20Keuangan%20Tahun%20{TAHUN}/{PERIODE}/{KODE}/instance.zip
```

dengan `{PERIODE}` ∈ `TW1|TW2|TW3|Audit` (verified: `Tahun 2026/TW1/BBCA/instance.zip`).
Path ini tetap di belakang Cloudflare — butuh TLS profile Firefox yang sama.
`// TODO(verify):` apakah periode audit memakai folder `Audit` atau `TW4`; cek via
`GetFinancialReport` daripada menebak path.

## 3. Struktur XBRL

Sumber: `instance.zip` BBCA TW1 2026 → berisi `instance.xbrl` (~1.8 MB) + `Taxonomy.xsd` (~18 KB,
hanya schemaRef lokal). Dokumen dihasilkan **Fujitsu Interstage XWand** — seragam karena semua emiten
memakai tooling pelaporan yang sama dari IDX.

- **Namespace**: `idx-cor` (akun keuangan) dan `idx-dei` (metadata entitas), taksonomi
  `http://www.idx.co.id/xbrl/taxonomy/2020-01-01/{cor,dei}`.
- **Konteks standar** (id konsisten): `CurrentYearInstant` (tanggal neraca periode ini),
  `PriorEndYearInstant` (akhir tahun lalu), `CurrentYearDuration` / `PriorYearDuration` (periode P&L
  YTD). Konteks berdimensi (segmen/komponen ekuitas) punya suffix `_..._<Member>` dan membawa
  `xbrldi:explicitMember` — **harus difilter** agar dapat angka konsolidasi, bukan breakdown.
- **Unit & presisi**: `unitRef="IDR"` nilai rupiah penuh, `decimals="-6"` (presisi jutaan);
  EPS memakai unit `IDRPerShares`.
- **Metadata** dari `idx-dei`: `EntityName`, `EntityCode`, `EntityMainIndustry`,
  `PeriodOfFinancialStatementsSubmissions`, `CurrentPeriodEndDate` — cukup untuk identifikasi
  dokumen tanpa parsing nama file.

### Hasil ekstraksi `spike-xbrl` (BBCA, Q1 2026, konsolidasi)

| Akun (idx-cor) | Q1 2026 | Pembanding |
|---|---:|---:|
| Assets | 1.640,83 T | 1.586,83 T (31 Des 2025) |
| Liabilities | 1.370,36 T | 1.294,51 T |
| Equity | 259,36 T | 281,69 T |
| EquityAttributableToEquityOwnersOfParentEntity | 259,13 T | 281,47 T |
| InterestIncome | 24,59 T | 24,37 T (Q1 2025) |
| ProfitLoss | 14,69 T | 14,15 T |
| ProfitLossAttributableToParentEntity | 14,68 T | 14,15 T |
| ComprehensiveIncome | 13,30 T | 14,51 T |
| BasicEarningsLossPerShareFromContinuingOperations | 119 | 115 |

9 akun kunci berhasil diekstrak (target minimal 5). Parser: `cmd/spike-xbrl/main.go`, streaming
`encoding/xml` murni tanpa dependency eksternal, memory flat.

- **Mudah diekstrak**: akun headline neraca & laba rugi di atas — nama konsep stabil dan konteksnya baku.
- **Perlu hati-hati**: `CashAndCashEquivalents` tidak muncul di BBCA (bank memakai konsep kas
  spesifik industri); banyak konsep hanya ada per tipe industri.
- `// TODO(verify):` keseragaman antar sektor baru dicek untuk 1 emiten (BBCA, bank — statement code
  4220000). Emiten non-keuangan (mis. TLKM, ASII) kemungkinan memakai `Revenue*`/`GrossProfit` alih-alih
  `InterestIncome`, dan statement code berbeda. Perlu dicek 2–3 emiten lintas sektor di awal Phase 2
  (sisa budget request spike sudah menipis).

## 4. Rekomendasi Arsitektur Fetcher (Phase 2)

- **HTTP client**: satu interface `Fetcher` dengan implementasi `tlsFetcher` berbasis
  `bogdanfinn/tls-client`, profile Firefox terbaru sebagai default, fallback rotasi ke profil Firefox
  lain saat 403. Header wajib: User-Agent Firefox yang konsisten dengan profile TLS, `Accept`,
  `Accept-Language`, `Referer: https://www.idx.co.id/`.
- **Rate limiting**: limiter global (bukan per-endpoint) min 15 detik antar request, exponential
  backoff + jitter saat 403/5xx, circuit breaker setelah N kegagalan beruntun (hindari memancing
  hard-ban IP).
- **Deteksi challenge**: perlakukan response non-JSON (Content-Type `text/html` atau body diawali
  `<!DOCTYPE`) sebagai kegagalan Cloudflare, bukan data — jangan di-cache.
- **TTL cache SQLite per jenis data**:
  - Trading/OHLCV + foreign flow: TTL ~1 hari (refresh setelah jam tutup bursa); data historis immutable.
  - Company profile: TTL 30 hari.
  - Financial report (XBRL): immutable per (emiten, tahun, periode) — cache selamanya, hanya cek
    ketersediaan periode baru per kuartal.
- **Parser XBRL**: pertahankan pendekatan streaming `encoding/xml` + whitelist konsep per tipe
  industri; jangan tarik library XBRL penuh.
- **Legal/etika**: fetch on-demand per user, tanpa redistribusi; rate limit konservatif hardcoded.

## 5. Daftar Risiko: Tervalidasi vs Terbantahkan

**Tervalidasi:**
- Cloudflare BEI agresif: `net/http` polos dan TLS profile Chrome 100% diblokir (403).
- Probing buta endpoint tidak efisien: `GetDividend`/`GetCorporateAction` 503, parameter
  `GetFinancialReport` tidak bisa ditebak (butuh DevTools → `reportType=rdf`).
- Ketersediaan konsep XBRL bergantung tipe industri (risiko untuk normalisasi lintas emiten).

**Terbantahkan:**
- ~~Perlu headless browser/Playwright~~ → TLS fingerprint Firefox saja cukup; single-binary Go tetap feasible.
- ~~Prefix `/umbraco/Surface/` masih dipakai~~ → situs baru memakai `/primary/`; endpoint lama tidak relevan.
- ~~Data XBRL sulit diparse~~ → format sangat teratur (generator tunggal Fujitsu XWand), parser stdlib cukup.

**Risiko terbuka (belum tervalidasi):**
- Cloudflare bisa menutup celah profil Firefox kapan saja → perlu strategi degradasi (pesan error
  jelas + opsi user menyuplai cookie sendiri).
- Keseragaman XBRL antar sektor belum terbukti (baru 1 emiten).
- Endpoint dividen/corporate action belum ditemukan.

## Definition of Done Phase 1 — Status

- [x] `spike-trading` print harga + foreign flow BBCA dari endpoint resmi
- [x] Kepastian client: `net/http` tidak cukup; wajib `tls-client` profile Firefox
- [x] ≥2 endpoint tambahan terkatalog (`GetCompanyProfilesDetail`, `GetFinancialReport` + path statis instance.zip; dividen/corp action terkonfirmasi butuh sesi DevTools)
- [x] `spike-xbrl` ekstrak 9 akun kunci dari laporan keuangan BBCA Q1 2026
- [x] Dokumen ini sebagai input Phase 2

package main

// golden - отчёт исходной версии Analyze на GenerateLog(20000).
// Не меняй его: оптимизация не должна менять результат.
const golden = `период: 2026-10-23T10:00:00Z - 2026-10-23T10:03:19Z, пропущено строк: 16
DELETE /api/sessions/:id       count=2546   5xx=70    p95=20ms
GET /api/orders/:id/items/:id  count=2501   5xx=50    p95=99ms
GET /api/products              count=2549   5xx=45    p95=119ms
GET /api/products/:id          count=2527   5xx=55    p95=40ms
GET /api/users/:id             count=2565   5xx=67    p95=59ms
GET /api/users/:id/orders      count=2475   5xx=41    p95=158ms
GET /healthz                   count=2405   5xx=45    p95=4ms
POST /api/orders               count=2432   5xx=61    p95=471ms
`

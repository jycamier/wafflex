# Changelog

## 0.0.1 (2026-03-16)


### Features

* **baseline:** add baseline management for stable diff reference ([dab1bcc](https://github.com/jycamier/wafflex/commit/dab1bccb2b74d630666d30fc33fb91f59e479586))
* **cache:** add TTL-based expiry for query cache and analysis results ([5631f56](https://github.com/jycamier/wafflex/commit/5631f56f1112cacae65fa85c01af6828bc7e9e4f))
* **cache:** clear command also removes analysis results ([fb5ebdf](https://github.com/jycamier/wafflex/commit/fb5ebdf3628a708a9040b2bdd2fd7a9840ab101a))
* **cmd:** add status command ([36fabf7](https://github.com/jycamier/wafflex/commit/36fabf77c6198f1df1f1cb8bcdddc44608fa3d23))
* initial commit ([b397fdd](https://github.com/jycamier/wafflex/commit/b397fdd81636601bc4b4682a458a0b46a532eb68))
* **parser:** add timestamp column mapping for parquet sources ([a16a54b](https://github.com/jycamier/wafflex/commit/a16a54b024895cd66ec1019d829733b52b28ae99))
* **scripts:** add k6 WAF test script and clean up load script ([0a4304e](https://github.com/jycamier/wafflex/commit/0a4304e0cacc58b485e8844776440de0def707cd))
* **version:** resolve version from debug build info ([77d2bd8](https://github.com/jycamier/wafflex/commit/77d2bd87e9bb94c93ecc83f1dada1d665cea97fe))
* **waf:** add rules for headers, body, URI and SSRF coverage ([af3a13a](https://github.com/jycamier/wafflex/commit/af3a13aad4282d286094c5ba06123a952f689c8d))


### Bug Fixes

* **baseline:** handle - as argument instead of subcommand ([f3abae5](https://github.com/jycamier/wafflex/commit/f3abae59c4dc52152ced29be47439a0d7613d8a4))

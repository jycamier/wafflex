# OWASP Top 10 HTTP Attack Templates

Ce dossier contient des templates HTTP pour tester les vulnérabilités OWASP Top 10 (2021).

## Structure

- `01-broken-access-control.http` - A01:2021 Broken Access Control
- `02-cryptographic-failures.http` - A02:2021 Cryptographic Failures
- `03-injection.http` - A03:2021 Injection
- `04-insecure-design.http` - A04:2021 Insecure Design
- `05-security-misconfiguration.http` - A05:2021 Security Misconfiguration
- `06-vulnerable-components.http` - A06:2021 Vulnerable and Outdated Components
- `07-authentication-failures.http` - A07:2021 Identification and Authentication Failures
- `08-integrity-failures.http` - A08:2021 Software and Data Integrity Failures
- `09-logging-failures.http` - A09:2021 Security Logging and Monitoring Failures
- `10-ssrf.http` - A10:2021 Server-Side Request Forgery (SSRF)
- `bonus-xss.http` - XSS attacks (complément)

## Format

Les fichiers utilisent le format HTTP standard avec variables :

```http
### Description de l'attaque
GET /path?param=payload HTTP/1.1
Host: {{host}}
Header: value

Body content
```

## Variables

- `{{host}}` - Remplacer par votre cible (ex: `localhost:8888`)

## Utilisation

### Avec curl

```bash
# Extraire et envoyer une requête
cat templates/03-injection.http | grep -A 10 "SQL Injection - UNION" | \
  sed 's/{{host}}/localhost:8888/' | \
  curl -K -
```

### Avec le script de génération

```bash
# Créer un script qui parse les templates
./generate-from-templates.sh templates/ http://localhost:8888
```

### Avec wafflex

```bash
# Analyser les templates avec Coraza
./wafflex test-templates -d templates/ -c coraza-test.conf
```

## Avertissement

⚠️ **Ces templates contiennent des payloads malveillants réels.**

- À utiliser UNIQUEMENT sur vos propres systèmes de test
- Ne JAMAIS utiliser sur des systèmes en production
- Ne JAMAIS utiliser sans autorisation explicite
- Usage illégal = sanctions pénales

## Références

- [OWASP Top 10 2021](https://owasp.org/Top10/)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [PayloadsAllTheThings](https://github.com/swisskyrepo/PayloadsAllTheThings)

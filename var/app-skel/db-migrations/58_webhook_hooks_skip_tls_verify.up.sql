-- Incoming message hooks verify TLS certificates unless the hook opts out,
-- e.g. for an endpoint with a self-signed certificate.
ALTER TABLE `webhook_hooks`
  ADD COLUMN `skip_tls_verify` tinyint(1) NOT NULL DEFAULT 0 AFTER `enabled`;

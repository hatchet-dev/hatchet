-- name: GetEvent :one
SELECT e.*
FROM v1_event_lookup_table elt
JOIN v1_event e ON (elt.event_id, elt.event_seen_at, elt.tenant_id) = (e.id, e.seen_at, e.tenant_id)
WHERE
    elt.external_id = $1
;

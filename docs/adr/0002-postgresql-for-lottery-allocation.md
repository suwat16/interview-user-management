# Keep lottery search and allocation in PostgreSQL

Use PostgreSQL as the single authority for indexed candidate lookup, ticket ownership, and request deduplication. A separate search engine would offer independent query scaling but require reconciliation of stale candidates with the allocation database, while PostgreSQL lets one transaction reserve the ticket and record the response. Including eligibility time in the lookup indexes makes expired tickets reusable immediately and adds index writes on each reservation.

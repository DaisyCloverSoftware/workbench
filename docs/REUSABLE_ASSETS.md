# Versioned reusable assets

Workbench memory records facts and decisions. Reusable assets go one step further: they preserve a routine or code payload as a versioned object that can be retrieved and reused instead of rebuilt.

Each asset belongs to a stable series and has immutable revisions. The latest revision is returned by normal search; history remains available for provenance and rollback. Assets may be project-scoped or global, but project assets never become global implicitly.

An asset records its kind (`routine` or `code`), name, summary, exact reusable content, optional language/tags, source provenance, a content digest, verification metadata, and usage counters. Saving identical content is idempotent. Saving changed content creates the next revision in the same series.

Secrets are rejected. Search remains advisory: the current repository is authoritative, and a worker must verify an asset still fits before applying it.

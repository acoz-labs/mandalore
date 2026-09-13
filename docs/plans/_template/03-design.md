# Technical Design

## Component And Behavior Flow

Use the smallest useful Mermaid diagrams. Cover entry, success, validation,
denial, failure, retry or recovery, asynchronous work, and visible state
transitions. Explain semantics the diagram cannot express.

## State And Data Model

Describe entities, relationships, lifecycle, ownership, constraints, indexes,
migration/backfill, compatibility, deletion/retention, and rollback. Use a
Mermaid ERD when relationships materially aid review.

## Interfaces And Contracts

Define commands, queries, endpoints, events, jobs, components, inputs, outputs,
errors, idempotency, transactions, and compatibility without method bodies.
Trace interfaces to the flow and data invariants.

## Authorization And Data Exposure

For each surface, name subject, action, resource, scope or condition, decision,
denial behavior, audit evidence, credential boundary, and public/frontend
visibility. Apply least privilege and cite existing enforcement precedent.

## Failure, Recovery, And Observability

State partial-failure behavior, retry rules, conflict handling, audit/logging
constraints, operational diagnosis, and escalation paths.

## Design Traceability

Show that every acceptance criterion and critical journey has a corresponding
component, state, interface, authority boundary, and recovery path.

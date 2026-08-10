# ADR 0001: Keep a modular monolith

**Status**: Accepted
**Date**: 2026-08-10

## Context

Boardit needs clearer ownership for notes, authentication, AI candidates, and asynchronous work. Its current team size, traffic, and deployment model do not justify distributed-service operations.

## Decision

Keep one Go API and one PostgreSQL database. Enforce internal direction through HTTP adapters, application use-cases, repository interfaces, and persistence adapters. Deploy with the existing React frontend and Docker Compose stack.

## Consequences

- Transactions and local development remain simple.
- AI and worker boundaries are expressed as modules and tables, not services.
- A module may be extracted later only with measured scaling, isolation, or ownership evidence.
- Cross-module shortcuts—especially direct GORM writes—are prohibited because they would erase the future extraction boundary.

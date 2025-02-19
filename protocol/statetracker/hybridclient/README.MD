# Hybrid Client

This package provides a hybrid client implementation that maintains compatibility between Cosmos SDK v47 and v50 event handling.

## Background

In Cosmos SDK v50, there were significant changes to block event handling:
- `BeginBlockEvents` and `EndBlockEvents` were deprecated
- A new event type `FinalizeBlockEvents` was introduced to replace them

This hybrid client seamlessly handles both versions by:
- Supporting the legacy `BeginBlockEvents` and `EndBlockEvents` for v47 compatibility
- Supporting the new `FinalizeBlockEvents` for v50 compatibility
- Aggregating all events into `FinalizeBlockEvents` for consistent downstream handling

This allows applications to work with both v47 and v50 chains without needing version-specific event handling logic.

---
targets: [code]
include_source: false
granularity: file
description: '-'
---
Tests are table-driven and mock-free

- When there is more than one test case, or the same logic is exercised repeatedly, use a table-driven test. "This case is different" is not grounds for a separate ad-hoc test unless the logic under test genuinely differs.
- Use test helpers and real implementations with test data instead of mocks.
- If code cannot be tested without mocking, treat that as a design defect and fix the design.

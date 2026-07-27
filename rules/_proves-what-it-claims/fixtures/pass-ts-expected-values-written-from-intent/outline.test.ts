import { describe, it, expect } from "vitest";

import { buildOutline, renderOutline } from "./outline";

const DOC = `# Onboarding

Everything a new workspace needs on its first day.

## Creating an account

## Inviting your team

### Roles and permissions

Owners may change billing; members may not.

### Seat limits

# Billing

## Plans

### Annual discounts
`;

const EXPECTED_OUTLINE = `Onboarding
  Creating an account
  Inviting your team
    Roles and permissions
    Seat limits
Billing
  Plans
    Annual discounts`;

describe("buildOutline", () => {
  it("nests each heading under the nearest shallower heading", () => {
    expect(buildOutline(DOC)).toEqual([
      {
        title: "Onboarding",
        level: 1,
        children: [
          { title: "Creating an account", level: 2, children: [] },
          {
            title: "Inviting your team",
            level: 2,
            children: [
              { title: "Roles and permissions", level: 3, children: [] },
              { title: "Seat limits", level: 3, children: [] },
            ],
          },
        ],
      },
      {
        title: "Billing",
        level: 1,
        children: [
          {
            title: "Plans",
            level: 2,
            children: [
              { title: "Annual discounts", level: 3, children: [] },
            ],
          },
        ],
      },
    ]);
  });
});

describe("renderOutline", () => {
  it("indents each heading by its depth", () => {
    expect(renderOutline(buildOutline(DOC))).toBe(EXPECTED_OUTLINE);
  });
});

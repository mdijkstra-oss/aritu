import { describe, it, expect } from "vitest";

import { withRetry } from "./retry";

describe("withRetry", () => {
  it("retries once after a failure and returns the second result", async () => {
    const fetchRate = async (): Promise<string> => "1.0942";

    const result = await withRetry(fetchRate, { attempts: 3, backoffMs: 10 });

    expect(result).toBe("1.0942");
  });

  it("waits twice as long before each further attempt", async () => {
    let calls = 0;
    const fetchRate = async (): Promise<string> => {
      calls += 1;
      if (calls === 1) {
        throw new Error("rate service unavailable");
      }
      return "1.0942";
    };

    const result = await withRetry(fetchRate, { attempts: 3, backoffMs: 10 });

    expect(result).toBe("1.0942");
  });
});

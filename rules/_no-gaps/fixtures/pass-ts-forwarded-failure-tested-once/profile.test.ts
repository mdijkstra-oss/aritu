import { describe, it, expect } from "vitest";

import { displayName, type User, type UserStore } from "./profile";

function storeOf(users: Record<string, User>): UserStore {
  return {
    async load(id: string): Promise<User> {
      const user = users[id];
      if (user === undefined) {
        throw new Error(`no user with id ${id}`);
      }

      return user;
    },
  };
}

describe("displayName", () => {
  it("uses the nickname when the user has one", async () => {
    const store = storeOf({ "u-1": { nickname: "ada", first: "Ada", last: "Lovelace" } });

    await expect(displayName(store, "u-1")).resolves.toBe("ada");
  });

  it("falls back to the first and last name when the nickname is absent", async () => {
    const store = storeOf({ "u-2": { nickname: null, first: "Grace", last: "Hopper" } });

    await expect(displayName(store, "u-2")).resolves.toBe("Grace Hopper");
  });

  it("lets a failure from the store reach the caller", async () => {
    const store = storeOf({});

    await expect(displayName(store, "u-7")).rejects.toThrow("no user with id u-7");
  });
});

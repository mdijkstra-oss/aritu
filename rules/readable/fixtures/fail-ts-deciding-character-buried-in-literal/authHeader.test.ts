import { describe, it, expect } from "vitest";

import { parseBearerToken } from "./authHeader";

describe("parseBearerToken", () => {
  it("rejects a token containing a character outside base64url", () => {
    const header =
      "Bearer 2H_EAPWMelNmdXewR32K8O2px0R0lI4RVsfZqtRD-FTClhRIstvs5aOAa5L-wQDXY44VGPZNfGp3cm_NhWWR8002D-uSPh-G_EROriF7dmfpb4sMCflnUqZTj-r-j-hmmgUVmiQTVWCiBxSHCR9sbEzpE0YNa+eG9HVFBTU40MEdpXzrNYKg_LYcetOFEjBtArm5ptAeC1raSy5z-zQr84_yrqAGc4pQT8QC7bA-Q24ZAcaq";

    expect(parseBearerToken(header)).toEqual({ error: "invalid token character" });
  });
});

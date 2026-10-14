import config from "./session-isolation.config";

export default {
  ...config,
  projects: [
    { name: "production-session", testMatch: "session-isolation.spec.ts" },
    {
      name: "prefixed-owner-member",
      testMatch: "session-isolation.spec.ts",
      grep: /owner → member/,
      metadata: { apiBase: "/gateway/api/v1" },
    },
    {
      name: "worker-security",
      testMatch: ["worker-api-isolation.spec.ts", "coach-isolation.spec.ts"],
      grepInvert: /real API availability/,
      repeatEach: 3,
    },
  ],
};

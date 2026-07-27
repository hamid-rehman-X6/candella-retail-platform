// Shared base ESLint rules for the monorepo.
// Individual apps/packages spread this array and append their own
// framework-specific configs (e.g. Next.js's `next/core-web-vitals`).
const baseConfig = [
  {
    ignores: [
      "**/dist/**",
      "**/.next/**",
      "**/node_modules/**",
      "**/.turbo/**",
    ],
  },
  {
    rules: {
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_" },
      ],
    },
  },
];

export default baseConfig;

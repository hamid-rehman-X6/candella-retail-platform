/**
 * Root Prettier config for the whole monorepo (frontend + shared packages + docs).
 * The Go backend is formatted by `gofmt`, not Prettier (see .prettierignore).
 *
 * @type {import("prettier").Config}
 */
const config = {
  semi: true,
  singleQuote: false,
  trailingComma: "all",
  printWidth: 80,
  tabWidth: 2,
  // Keep hand-wrapped markdown prose as-is; only normalise structure/tables.
  proseWrap: "preserve",

  // Auto-sort Tailwind classes. Must be the last plugin.
  plugins: ["prettier-plugin-tailwindcss"],
  // Tailwind v4 is CSS-first: point the plugin at the stylesheet that holds
  // the @theme tokens so it understands our custom utilities.
  tailwindStylesheet: "./frontend/app/globals.css",
  // Also sort classes passed to these helper functions, not just className.
  tailwindFunctions: ["cn", "clsx", "cva"],
};

export default config;

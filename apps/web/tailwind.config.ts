import type { Config } from "tailwindcss";
import defaultTheme from "tailwindcss/defaultTheme";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["var(--font-sans)", ...defaultTheme.fontFamily.sans],
        mono: ["var(--font-mono)", ...defaultTheme.fontFamily.mono],
      },
      borderRadius: {
        // Make the whole UI feel sharper - default rounded-lg now renders at 4px
        // instead of 8px, rounded-md at 3px, rounded at 2px, rounded-sm at 1px.
        sm: "1px",
        DEFAULT: "2px",
        md: "3px",
        lg: "4px",
      },
    },
  },
  plugins: [],
};

export default config;

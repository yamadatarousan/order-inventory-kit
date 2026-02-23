/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        paper: "#fbfaf6",
        ink: "#1f2937",
        brand: {
          50: "#e8fbf5",
          100: "#d0f5ea",
          600: "#0f766e",
          700: "#0d5f59",
          900: "#0b3f3b",
        },
      },
      boxShadow: {
        card: "0 22px 48px -24px rgba(15, 23, 42, 0.45)",
      },
      fontFamily: {
        heading: ["Outfit", "Avenir Next", "Noto Sans JP", "sans-serif"],
        body: ["BIZ UDPGothic", "Hiragino Kaku Gothic ProN", "sans-serif"],
      },
    },
  },
  plugins: [],
};

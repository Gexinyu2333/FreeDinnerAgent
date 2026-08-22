/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: {
          50: "#f8fafc",
          100: "#eef2f7",
          200: "#d9e2ec",
          500: "#64748b",
          700: "#334155",
          900: "#0f172a"
        },
        ocean: {
          500: "#2563eb",
          600: "#1d4ed8"
        },
        mint: {
          500: "#10b981",
          600: "#059669"
        },
        amber: {
          500: "#f59e0b"
        }
      },
      boxShadow: {
        soft: "0 10px 30px rgba(15, 23, 42, 0.08)"
      }
    }
  },
  plugins: []
};
